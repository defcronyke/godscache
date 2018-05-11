// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

// Package godscache is a wrapper around the official Google Cloud Datastore Go client library
// "cloud.google.com/go/datastore", which adds caching to datastore requests. Its name is a play on words,
// and it's actually called Go DS Cache. Note that it is wrapping the newer datastore library, which is
// intended for use with App Engine Flexible Environment, Compute Engine, or Kubernetes Engine, and is not
// for use with App Engine Standard.
//
// If you're looking for something similar for App Engine Standard, check out the highly recommended nds library:
// https://godoc.org/github.com/qedus/nds
//
// For the offical Google library documentation, go here: https://godoc.org/cloud.google.com/go/datastore
//
// Godscache is designed to allow concurrent datastore requests, and it follows the Google Cloud Datastore client
// library API very closely, to act as a drop-in replacement for the official library from Google.
//
// Things which aren't implemented in this library can be used anyway, they just won't be cached.
// For example, the Client struct holds a Parent member which is the raw Google Datastore client,
// so you can use that instead to make your requests if you need to use some feature that's not implemented
// in godscache, or if you want to bypass the cache for some reason.
package godscache

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"reflect"
	"strconv"
	"sync"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/option"
)

// Client is the main struct for godscache. It holds a regular datastore client in the Parent field, as well as the cache and max cache size.
type Client struct {
	Parent       *datastore.Client      // The regular datastore client, which can be used directly if you want to bypass caching.
	ProjectID    string                 // The Google Cloud Platform project ID.
	Cache        map[string]interface{} // The application-level cache.
	MaxCacheSize int                    // Cache size in number of items.
	cacheKeys    []string               // A slice of all the keys in the cache. Used to determine which entries to evict when the cache is full.
	cacheMx      *sync.RWMutex          // A mutex to support accessing the cache concurrently.
	cacheKeysMx  *sync.RWMutex          // A mutex to support accessing the cache keys concurrently.
}

// NewClient is a constructor for making a new godscache client. Start here. It makes a datastore client and stores it in the Parent field.
// The max cache size defaults to 1000 items. To change that, set the GODSCACHE_MAX_CACHE_SIZE environment variable before running this function.
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	// Create datastore client.
	dsClient, err := datastore.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	// Set max cache size in number of items.
	maxCacheSize := 1000
	maxCacheSizeStr := os.Getenv("GODSCACHE_MAX_CACHE_SIZE")
	if maxCacheSizeStr != "" {
		maxCacheSize64, err := strconv.ParseInt(maxCacheSizeStr, 10, 32)
		if err != nil {
			return nil, err
		}
		maxCacheSize = int(maxCacheSize64)
	}

	// Instantiate a new godscache Client and return a pointer to it.
	c := &Client{
		Parent:       dsClient,
		ProjectID:    projectID,
		Cache:        make(map[string]interface{}, maxCacheSize),
		cacheMx:      &sync.RWMutex{},
		MaxCacheSize: maxCacheSize,
		cacheKeys:    make([]string, 0, maxCacheSize),
		cacheKeysMx:  &sync.RWMutex{},
	}

	return c, nil
}

// Run a datastore query. To utilize this with caching, you should perform a KeysOnly() query, and then use Get() on the keys.
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
		return nil, err
	}

	// Put data into the cache, indexed by the string representation of the datastore key.
	keyStr := key.String()
	c.cacheMx.Lock()
	c.Cache[keyStr] = src
	c.cacheMx.Unlock()

	// Put the datastore key string into the slice of cache keys.
	c.cacheKeysMx.Lock()
	c.cacheKeys = append(c.cacheKeys, keyStr)
	c.cacheKeysMx.Unlock()

	// Get the number of elements in the cache keys slice. This should be the same as the number of items in the cache.
	c.cacheKeysMx.RLock()
	lenCacheKeys := len(c.cacheKeys)
	c.cacheKeysMx.RUnlock()

	// IF the cache is full, remove the oldest item from the cache and from the slice of cache keys.
	if lenCacheKeys > c.MaxCacheSize {
		c.cacheMx.Lock()
		c.cacheKeysMx.RLock()
		delete(c.Cache, c.cacheKeys[0])
		c.cacheKeysMx.RUnlock()
		c.cacheMx.Unlock()

		c.cacheKeysMx.Lock()
		c.cacheKeys = c.cacheKeys[1:]
		c.cacheKeysMx.Unlock()
	}

	return key, nil
}

// Get data from the datastore or cache. The dst value must be a Struct pointer.
func (c *Client) Get(ctx context.Context, key *datastore.Key, dst interface{}) error {
	// Get data from the cache if it's in there.
	keyStr := key.String()
	c.cacheMx.RLock()
	cacheDst, cached := c.Cache[keyStr]
	c.cacheMx.RUnlock()

	// Check if the requested data wasn't found in the cache.
	if !cached {
		// Get data from the datastore, and save it in dst.
		err := c.Parent.Get(ctx, key, dst)
		if err != nil {
			return err
		}

		// Put data into the cache.
		log.Printf("Cache MISS while running Get(): %+v", dst)
		c.cacheMx.Lock()
		c.Cache[keyStr] = dst
		c.cacheMx.Unlock()

		// Put the datastore key string into the slice of keys.
		c.cacheKeysMx.Lock()
		c.cacheKeys = append(c.cacheKeys, keyStr)
		c.cacheKeysMx.Unlock()
	} else {
		// If the requested data was cached, convert it from interface to its correct type.
		log.Printf("Cache HIT while running Get(): %+v", cacheDst)
		cVal := reflect.ValueOf(cacheDst)
		dVal := reflect.ValueOf(dst)

		// Make sure dst is a pointer.
		if dVal.Kind() != reflect.Ptr {
			return errors.New("dst has a different type than what's in the cache")
		}

		// Get the type names.
		dstName := reflect.TypeOf(dst).String()
		cDstName := reflect.TypeOf(cacheDst).String()

		// Make sure the data from cache is the same type as dst.
		if dstName != cDstName {
			return fmt.Errorf("dVal and cVal are not the same struct. dstName: %v : cDstName: %v", dstName, cDstName)
		}

		// Save data into dst.
		cVal = cVal.Elem()
		dVal = dVal.Elem()
		dVal.Set(cVal)
	}

	return nil
}

// GetMulti is for getting multiple values from the datastore or cache.
// The dst value must be a slice of structs or struct pointers, and not a datastore.PropertyList.
func (c *Client) GetMulti(ctx context.Context, keys []*datastore.Key, dst interface{}) error {
	dVal := reflect.ValueOf(dst)
	dstType := reflect.TypeOf(dst)
	dstName := dstType.String()

	if dVal.Kind() != reflect.Slice {
		return errors.New("dst must be a slice of structs or struct pointers")
	}

	if dstName == "datastore.PropertyList" {
		return errors.New("dst must not be a datastore.PropertyList")
	}

	if len(keys) != dVal.Len() {
		return errors.New("keys and dst must be the same length")
	}

	uncachedKeys := append([]*datastore.Key(nil), keys...)
	cachedKeys := make([]*datastore.Key, 0, len(keys))
	resultsMap := make(map[string]interface{}, len(keys))

	for idx, key := range keys {
		keyStr := key.String()

		c.cacheMx.RLock()
		val, cached := c.Cache[keyStr]
		c.cacheMx.RUnlock()

		if idx >= len(uncachedKeys) {
			c.cacheMx.RLock()
			resultsMap[keyStr] = val
			c.cacheMx.RUnlock()
			break
		}

		if cached {
			if len(uncachedKeys) > 1 {
				uncachedKeys = append(uncachedKeys[:idx], uncachedKeys[idx+1:]...)
			}

			cachedKeys = append(cachedKeys, key)

			c.cacheMx.RLock()
			resultsMap[keyStr] = val
			c.cacheMx.RUnlock()
		}
	}

	if len(uncachedKeys) > 0 {
		dsResultsSlice := reflect.MakeSlice(dstType, len(uncachedKeys), len(uncachedKeys))
		dsResults := reflect.New(reflect.TypeOf(dst)).Elem()
		dsResults.Set(dsResultsSlice)

		err := c.Parent.GetMulti(ctx, uncachedKeys, dsResults.Interface())
		if err != nil {
			return err
		}

		for idx, key := range uncachedKeys {
			keyStr := key.String()
			resultsMap[keyStr] = dsResults.Index(idx).Interface()
		}
	}

	results := make([]interface{}, 0, len(keys))

	for _, key := range keys {
		keyStr := key.String()
		results = append(results, resultsMap[keyStr])
	}

	for idx, val := range results {
		dVal.Index(idx).Set(reflect.ValueOf(val))
	}

	return nil
}

// Delete data from the datastore and cache.
func (c *Client) Delete(ctx context.Context, key *datastore.Key) error {
	// Check if the data is cached.
	keyStr := key.String()
	c.cacheMx.RLock()
	_, cached := c.Cache[keyStr]
	c.cacheMx.RUnlock()

	if cached {
		// Delete data from cache.
		c.cacheMx.Lock()
		delete(c.Cache, keyStr)
		c.cacheMx.Unlock()

		// Delete key from cache keys slice.
		c.cacheKeysMx.Lock()
		for idx, val := range c.cacheKeys {
			if val == keyStr {
				if len(c.cacheKeys) > 1 {
					c.cacheKeys = append(c.cacheKeys[:idx], c.cacheKeys[idx+1:]...)
				} else {
					c.cacheKeys = make([]string, 0, c.MaxCacheSize)
				}
			}
		}
		c.cacheKeysMx.Unlock()
	}

	// Delete data from datastore.
	err := c.Parent.Delete(ctx, key)
	if err != nil {
		return err
	}

	return nil
}
