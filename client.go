package godscache

import (
	"context"
	"errors"
	"log"
	"os"
	"reflect"
	"strconv"
	"sync"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/option"
)

type Client struct {
	Parent       *datastore.Client      // The regular datastore client, which can be used directly if you want to bypass caching.
	Cache        map[string]interface{} // The application-level cache.
	MaxCacheSize int                    // Cache size in number of items.
	cacheKeys    []string               // A slice of all the keys in the cache. Used to determine which entries to evict when the cache is full.
	cacheMx      *sync.RWMutex          // A mutex to support accessing the cache concurrently.
	cacheKeysMx  *sync.RWMutex          // A mutex to support accessing the cache keys concurrently.
}

// NewClient is a constructor for making a new godscache client. Start here. It makes a datastore client and stores it in the Parent field.
// The max cache size defaults to 1000 items. To change that, set the GODSCACHE_MAX_CACHE_SIZE environment variable before running this function.
func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	dsClient, err := datastore.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	maxCacheSize := 1000
	maxCacheSizeStr := os.Getenv("GODSCACHE_MAX_CACHE_SIZE")
	if maxCacheSizeStr != "" {
		maxCacheSize64, err := strconv.ParseInt(maxCacheSizeStr, 10, 32)
		if err != nil {
			return nil, err
		}
		maxCacheSize = int(maxCacheSize64)
	}

	c := &Client{
		Parent:       dsClient,
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
	return c.Parent.Run(ctx, q)
}

// Put data into the datastore and into the cache. The src value must be a Struct pointer.
func (c *Client) Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error) {
	var err error
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		return nil, err
	}

	keyStr := key.String()
	c.cacheMx.Lock()
	c.Cache[keyStr] = src
	c.cacheMx.Unlock()

	c.cacheKeysMx.Lock()
	c.cacheKeys = append(c.cacheKeys, keyStr)
	c.cacheKeysMx.Unlock()

	c.cacheKeysMx.RLock()
	lenCacheKeys := len(c.cacheKeys)
	c.cacheKeysMx.RUnlock()

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
	keyStr := key.String()

	c.cacheMx.RLock()
	cacheDst, cached := c.Cache[keyStr]
	c.cacheMx.RUnlock()
	if !cached {
		err := c.Parent.Get(ctx, key, dst)
		if err != nil {
			return err
		}
		log.Printf("Cache MISS while running Get(): %+v", dst)
		c.cacheMx.Lock()
		c.Cache[keyStr] = dst
		c.cacheMx.Unlock()

		c.cacheKeysMx.Lock()
		c.cacheKeys = append(c.cacheKeys, keyStr)
		c.cacheKeysMx.Unlock()
	} else {
		log.Printf("Cache HIT while running Get(): %+v", cacheDst)
		cVal := reflect.ValueOf(cacheDst)
		dVal := reflect.ValueOf(dst)

		if dVal.Kind() != reflect.Ptr {
			return errors.New("dst has a different type than what's in the cache")
		}

		dstName := reflect.TypeOf(dst).String()
		cDstName := reflect.TypeOf(cacheDst).String()

		if dstName != cDstName {
			return errors.New("dVal and cVal are not the same struct")
		}

		cVal = cVal.Elem()
		dVal = dVal.Elem()

		dVal.Set(cVal)
	}

	return nil
}
