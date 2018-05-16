// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

// Package godscache is a wrapper around the official Google Cloud Datastore Go client library
// "cloud.google.com/go/datastore", which adds caching to datastore requests using memcached.
// Its name is a play on words, and it's actually called Go DS Cache. Note that it is wrapping
// the newer datastore library, which is intended for use with App Engine Flexible Environment,
// Compute Engine, or Kubernetes Engine, and is not for use with App Engine Standard.
//
// If you're looking for something similar for App Engine Standard, check out the highly recommended
// nds library: https://godoc.org/github.com/qedus/nds
//
// For the official Google library documentation, go here:
// https://godoc.org/cloud.google.com/go/datastore
//
// Godscache follows the Google Cloud Datastore client library API very closely, to act as a drop-in
// replacement for the official library from Google.
//
// Things which aren't implemented in this library can be used anyway, they just won't be cached.
// For example, the Client struct holds a Parent member which is the raw Google Datastore client,
// so you can use that instead to make your requests if you need to use some feature that's not
// implemented in godscache, or if you want to bypass the cache for some reason.
//
// To use godscache, you will need a Google Cloud project with an initialized Datastore on it,
// and a memcached instance to connect to. You can connect to as many memcached instances as you want.
package godscache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/bradfitz/gomemcache/memcache"
	"google.golang.org/api/option"
)

// Client is the main struct for godscache. It holds a regular datastore client in the
// Parent field, as well as the memcache client.
type Client struct {
	// The raw Datastore client, which can be used directly if you want to bypass caching.
	Parent *datastore.Client

	// The Google Cloud Platform project ID.
	ProjectID string

	// The memcached IP:PORT addresses.
	MemcacheServers []string

	// The memcache client, which you can use directly if you want to access the cache.
	MemcacheClient *memcache.Client
}

// NewClient is a constructor for making a new godscache client. Start here. It makes a datastore
// client and stores it in the Parent field, and it makes a memcache client. Set the context with
// the MemcacheServerKey, with a value of
// []string{"ip_address1:port", "ip_addressN:port"}, to specify which memcache servers to connect
// to. Alternately you can set the environment variable
// GODSCACHE_MEMCACHED_SERVERS="ip_address1:port,ip_addressN:port" instead to specify
// the memcached servers. The context value will take priority over the environment
// variables if both are present.
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	// Create datastore client.
	dsClient, err := datastore.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	// Get the list of memcached servers to connect to.
	memcacheServers := memcacheServers(ctx)

	// Create memcache client.
	memcacheClient := memcache.New(memcacheServers...)
	memcacheClient.Timeout = time.Second * 10
	memcacheClient.MaxIdleConns = 100

	// Instantiate a new godscache Client and return a pointer to it.
	c := &Client{
		Parent:          dsClient,
		ProjectID:       projectID,
		MemcacheServers: memcacheServers,
		MemcacheClient:  memcacheClient,
	}

	return c, nil
}

// Run a datastore query. To utilize this with caching, you should perform a KeysOnly() query,
// and then use Get() on the keys.
func (c *Client) Run(ctx context.Context, q *datastore.Query) *datastore.Iterator {
	// Perform the query using the datastore client.
	return c.Parent.Run(ctx, q)
}

// Put data into the datastore and into the cache. The src value must be a Struct pointer.
func (c *Client) Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error) {
	var err error

	// Put data into the datastore.
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		return nil, fmt.Errorf("godscache.Client.Put: failed putting src into datastore: %v", err)
	}

	// Add data to cache.
	err = c.addToCache(key, src)
	if err != nil {
		return nil, fmt.Errorf("godscache.Client.Put: failed adding item to cache: %v", err)
	}

	return key, nil
}

// PutMulti adds multiple pieces of data to the datastore and cache all at once.
// It returns a slice of complete keys.
func (c *Client) PutMulti(ctx context.Context, keys []*datastore.Key, src interface{}) ([]*datastore.Key, error) {
	// Put data into datastore.
	ret, err := c.Parent.PutMulti(ctx, keys, src)
	if err != nil {
		return nil, fmt.Errorf("godscache.Client.PutMulti: failed putting multiple entries into datastore: %v", err)
	}

	// Make a runtime value of the data.
	sVal := reflect.ValueOf(src)

	// Iterate over all the keys, adding the data to the cache.
	for idx, key := range keys {
		// Add data to the cache.
		err = c.addToCache(key, sVal.Index(idx).Interface())
		if err != nil {
			return nil, fmt.Errorf("godscache.Client.PutMulti: failed putting data into cache: %v", err)
		}
	}

	return ret, nil
}

// Get data from the datastore or cache. The dst value must be a Struct pointer.
func (c *Client) Get(ctx context.Context, key *datastore.Key, dst interface{}) error {
	// Get data from the cache if it's in there.
	cached := c.getFromCache(key, dst)

	// Check if the requested data wasn't found in the cache.
	if !cached {
		// Get data from the datastore, and save it in dst.
		err := c.Parent.Get(ctx, key, dst)
		if err != nil {
			return err
		}

		// Put data into the cache.
		// log.Printf("godscache.Client.Get: cache MISS: %v", key)
		err = c.addToCache(key, dst)
		if err != nil {
			return fmt.Errorf("godscache.Client.Get: failed adding item to cache: %v", err)
		}
	} else {
		// log.Printf("godscache.Client.Get: cache HIT: %v", key)
	}

	return nil
}

// GetMulti is for getting multiple values from the datastore or cache.
// The dst value must be a slice of structs or struct pointers, and not a datastore.PropertyList.
// It must also be the same length as the keys slice.
func (c *Client) GetMulti(ctx context.Context, keys []*datastore.Key, dst interface{}) error {
	// Get runtime value of dst.
	dVal := reflect.ValueOf(dst)

	// Get type of dst.
	dstType := reflect.TypeOf(dst)

	// Get string of dst type.
	dstName := dstType.String()

	// Make sure dst is of the coorect type and length.
	if dVal.Kind() != reflect.Slice {
		return errors.New("godscache.Client.GetMulti: dst must be a slice of structs or struct pointers")
	}

	if dstName == "datastore.PropertyList" {
		return errors.New("godscache.Client.GetMulti: dst must not be a datastore.PropertyList")
	}

	if len(keys) != dVal.Len() {
		return errors.New("godscache.Client.GetMulti: keys and dst must be the same length")
	}

	// Make some new data structures to hold keys and results.
	uncachedKeys := make([]*datastore.Key, 0)
	resultsMap := make(map[string]interface{}, len(keys))

	// Batch get items from cache.
	err := c.getMultiFromCache(keys, dst)
	if err != nil {
		return fmt.Errorf("godscache.Client.GetMulti: failed getting multiple items from cache: %v", err)
	}

	// log.Printf("godscache.Client.GetMulti: got multiple results from cache: %+v", dst)

	// For each key.
	for idx, key := range keys {
		// Check if we're missing the value because it wasn't in the cache.
		dVal2 := dVal.Index(idx)
		if (dVal2.Kind() == reflect.Ptr && dVal2.IsNil()) || dVal2.Kind() == reflect.Struct {
			// Add the key to the list of uncached keys, so we can use it below to request the Datastore.
			uncachedKeys = append(uncachedKeys, key)
		} else {
			// If the value was in the cache, add it to the results map.
			resultsMap[key.String()] = dVal2.Interface()
		}
	}

	// If there are any uncached keys, use them for a batch datastore lookup.
	if len(uncachedKeys) > 0 {
		// log.Printf("godscache.Client.GetMulti: number of cache misses: %v", len(uncachedKeys))

		// Make a new dynamic slice to hold the uncached results, that's the same length as the
		// uncached keys slice.
		dsResultsSlice := reflect.MakeSlice(dstType, len(uncachedKeys), len(uncachedKeys))

		// Make the slice addressable.
		dsResults := reflect.New(dstType).Elem()
		dsResults.Set(dsResultsSlice)

		// log.Printf("godscache.Client.GetMulti: dsResults type: %v", dsResults.Type().String())

		// Get the uncached data from the datastore.
		err := c.Parent.GetMulti(ctx, uncachedKeys, dsResults.Interface())
		if err != nil {
			return fmt.Errorf("godscache.Client.GetMulti: failed getting multiple values from datastore: %v", err)
		}

		// log.Printf("godscache.Client.GetMulti: dsResults: %+v", dsResults.Interface())

		// Add the data to the results map, and to the cache.
		for idx, key := range uncachedKeys {
			keyStr := key.String()

			res := dsResults.Index(idx).Interface()
			resultsMap[keyStr] = res

			err = c.addToCache(key, res)
			if err != nil {
				return fmt.Errorf("godscache.Client.GetMulti: failed adding item to cache: %v", err)
			}
		}
	}

	// Copy the results to dst in the correct order.
	for idx, key := range keys {
		keyStr := key.String()
		val, ok := resultsMap[keyStr]
		if !ok {
			return fmt.Errorf("godscache.Client.GetMulti: expected item not found in results map")
		}
		dVal.Index(idx).Set(reflect.ValueOf(val))
	}

	// log.Printf("godscache.Client.GetMulti: results: %+v", dst)

	return nil
}

// Delete data from the datastore and cache.
func (c *Client) Delete(ctx context.Context, key *datastore.Key) error {
	// Delete the data from the cache, if it's in there.
	err := c.deleteFromCache(key)
	if err != nil {
		return fmt.Errorf("godscache.Client.Delete: failed deleting item from cache: %v", err)
	}

	// Delete data from datastore.
	err = c.Parent.Delete(ctx, key)
	if err != nil {
		return fmt.Errorf("godscache.Client.Parent.Delete: failed deleting item from datastore: %v", err)
	}

	return nil
}

// DeleteMulti deletes multiple pieces of data from the datastore and cache all at once.
func (c *Client) DeleteMulti(ctx context.Context, keys []*datastore.Key) error {
	// Put data into datastore.
	err := c.Parent.DeleteMulti(ctx, keys)
	if err != nil {
		return fmt.Errorf("godscache.Client.DeleteMulti: failed deleting multiple entries from datastore: %v", err)
	}

	// Iterate over all the keys, deleting the data from the cache.
	for _, key := range keys {
		// Delete data from the cache.
		err = c.deleteFromCache(key)
		if err != nil {
			return fmt.Errorf("godscache.Client.DeleteMulti: failed deleting data from cache: %v", err)
		}
	}

	return nil
}

// Add an item to the cache.
func (c *Client) addToCache(key *datastore.Key, data interface{}) error {
	// Convert data to JSON bytes.
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("godscache.Client.addToCache: failed marshaling data to JSON: %v", err)
	}

	// Add JSON bytes to memcached server(s), indexed by the string representation of
	// the datastore key.
	err = c.MemcacheClient.Set(
		&memcache.Item{
			Key:   key.String(),
			Value: dataBytes,
		},
	)
	if err != nil {
		return fmt.Errorf("godscache.Client.addToCache: failed adding item to cache: %v", err)
	}

	return nil
}

// Get data from the cache, if it's in there. Returns true if there is a cache hit,
// and if so, it populates dst with the data. If there is a cache miss, dst is left
// untouched.
func (c *Client) getFromCache(key *datastore.Key, dst interface{}) bool {
	// Make sure dst is the right type.
	if dst == nil || reflect.ValueOf(dst).Kind() != reflect.Ptr {
		return false
	}

	// Try to get data from memcache server(s), and return false if the data isn't in there.
	item, err := c.MemcacheClient.Get(key.String())
	if err == memcache.ErrCacheMiss {
		return false
	}
	if err != nil {
		log.Printf("godscache.Client.getFromCache: failed getting data from memcached: %v", err)
		return false
	}

	// Load data into dst.
	err = json.Unmarshal(item.Value, dst)
	if err != nil {
		log.Printf("godscache.Client.getFromCache: failed unmarshaling JSON data from cache: %v", err)
	}

	return true
}

// Batch get data from the cache. The dst value must be a slice of pointers to structs,
// and must be the same length as the keys slice. The dst value will be populated with
// data if found in the cache, and nil for keys which aren't cached, in the order
// of the keys slice.
func (c *Client) getMultiFromCache(keys []*datastore.Key, dst interface{}) error {
	// Make the key strings slice, for use with memcache's get multi function.
	keyStrs := make([]string, 0, len(keys))
	for _, key := range keys {
		keyStrs = append(keyStrs, key.String())
	}

	// Batch get the data from memcached.
	items, err := c.MemcacheClient.GetMulti(keyStrs)
	if err != nil {
		return fmt.Errorf("godscache.Client.getMultiFromCache: failed getting multiple items from memcached: %v", err)
	}

	// Get the runtime value of dst.
	dVal := reflect.ValueOf(dst)

	// Insert the data into dst. It will skip inserting in positions where data wasn't found
	// in the cache, leaving those spots nil.
	for idx, key := range keys {
		// Check if data is cached, and if so, get it out of the cache.
		keyStr := key.String()
		item, cached := items[keyStr]
		if cached {
			// Create a new runtime value which can be unmarshalled into.
			dVal2 := reflect.New(reflect.TypeOf(dst).Elem())
			err = json.Unmarshal(item.Value, dVal2.Interface())
			if err != nil {
				return fmt.Errorf("godscache.Client.getMultiFromCache: failed unmarshaling cached data from JSON: %v", err)
			}

			// Copy the data into dst.
			dVal.Index(idx).Set(dVal2.Elem())
		}
	}

	return nil
}

// Delete data from cache.
func (c *Client) deleteFromCache(key *datastore.Key) error {
	// Delete data from memcached server(s).
	err := c.MemcacheClient.Delete(key.String())
	if err == memcache.ErrCacheMiss {
		return nil
	}
	if err != nil {
		return fmt.Errorf("godscache.deleteFromCache: failed deleting from memcache: %v", err)
	}

	return nil
}
