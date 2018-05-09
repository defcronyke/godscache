package godscache

import (
	"context"
	"errors"
	"log"
	"os"
	"reflect"
	"strconv"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/option"
)

type Client struct {
	Parent       *datastore.Client
	Cache        map[string]interface{}
	MaxCacheSize int // Size in number of items.
	cacheKeys    []string
}

// Max cache size defaults to 1000 items. To change that, set the
// GODSCACHE_MAX_CACHE_SIZE environment variable.
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
		MaxCacheSize: maxCacheSize,
		cacheKeys:    make([]string, 0, maxCacheSize),
	}

	return c, nil
}

func (c *Client) Run(ctx context.Context, q *datastore.Query) *datastore.Iterator {
	return c.Parent.Run(ctx, q)
}

func (c *Client) Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error) {
	var err error
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		return nil, err
	}

	keyStr := key.String()
	c.Cache[keyStr] = src

	c.cacheKeys = append(c.cacheKeys, keyStr)
	if len(c.cacheKeys) > c.MaxCacheSize {
		delete(c.Cache, c.cacheKeys[0])
		c.cacheKeys = c.cacheKeys[1:]
	}

	return key, nil
}

func (c *Client) Get(ctx context.Context, key *datastore.Key, dst interface{}) error {
	keyStr := key.String()

	cacheDst, cached := c.Cache[keyStr]
	if !cached {
		err := c.Parent.Get(ctx, key, dst)
		if err != nil {
			return err
		}
		log.Printf("Cache MISS while running Get(): %+v", dst)
		c.Cache[keyStr] = dst
		c.cacheKeys = append(c.cacheKeys, keyStr)
	} else {
		log.Printf("Cache HIT while running Get(): %+v", cacheDst)
		cVal := reflect.ValueOf(cacheDst)
		dVal := reflect.ValueOf(dst)

		if dVal.Kind() != reflect.Ptr {
			return errors.New("dst has a different type than what's in the cache")
		}

		dstName := reflect.TypeOf(dst).String()
		cDstName := reflect.TypeOf(cacheDst).String()

		log.Printf("dstName: %v", dstName)
		log.Printf("cDstName: %v", cDstName)

		if dstName != cDstName {
			return errors.New("dVal and cVal are not the same struct")
		}

		cVal = cVal.Elem()
		dVal = dVal.Elem()

		dVal.Set(cVal)
	}

	return nil
}
