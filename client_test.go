// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

// This is the test suite for godscache.Client.
//
// Set the environment variable GODSCACHE_PROJECT_ID to your Google Cloud Platform project ID before running these tests.
// It must be set to a valid GCP project ID of a project that you control, with an initialized datastore.
package godscache

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

type EmptyKind struct{}

type TestDbData struct {
	TestString string
}

type TestDbDataDifferent struct {
	TestInt int
}

// ----- Main -----

func TestMain(m *testing.M) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.TestMain: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		os.Exit(1)
	}

	err = c.MemcacheClient.DeleteAll()
	if err != nil {
		log.Printf("godscache.TestMain: deleting all data from memcache failed: %v", err)
		os.Exit(2)
	}

	res := m.Run()
	os.Exit(res)
}

// ----- End Main -----

// ----- Tests -----

func TestNewClientValidProjectID(t *testing.T) {
	ctx := context.Background()

	_, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}
}

func TestNewClientProjectIDEnvVar(t *testing.T) {
	os.Setenv("DATASTORE_PROJECT_ID", os.Getenv("GODSCACHE_PROJECT_ID"))

	ctx := context.Background()
	_, err := NewClient(ctx, "")
	if err != nil {
		t.Fatalf("Instantiating new Client struct with project ID in the DATASTORE_projectID environment variable failed: %v", err)
	}

	os.Unsetenv("DATASTORE_PROJECT_ID")
}

func TestNewClientNoProjectID(t *testing.T) {
	ctx := context.Background()

	_, err := NewClient(ctx, "")
	if err == nil {
		t.Fatalf("Instantiating new Client struct with no project ID succeeded.")
	}
}

func TestNewClientMemcachedContext(t *testing.T) {
	memcacheServers := strings.Split(os.Getenv("GODSCACHE_MEMCACHED_SERVERS"), ",")

	ctx := context.WithValue(context.Background(), MemcacheServerKey, memcacheServers)

	_, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("godscache.TestNewClientMemcachedContext: instantiating new Client struct failed: %v", err)
	}
}

func TestNewClientNoMemcached(t *testing.T) {
	ctx := context.Background()

	memcacheServers := os.Getenv("GODSCACHE_MEMCACHED_SERVERS")
	os.Unsetenv("GODSCACHE_MEMCACHED_SERVERS")

	_, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	os.Setenv("GODSCACHE_MEMCACHED_SERVERS", memcacheServers)
	if err != nil {
		t.Fatalf("godscache.TestNewClientNoMemcached: instantiating new Client with no memcached servers failed: %v", err)
	}
}

func TestRun(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	kind := "testRun"
	key := datastore.IncompleteKey(kind, nil)
	src := &TestDbData{TestString: "TestRun"}
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting test data into database: %v", err)
	}

	q := datastore.NewQuery(kind).Limit(1)
	for it := c.Run(ctx, q); ; {
		var res TestDbData
		_, err := it.Next(&res)
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("Failed running query: %v", err)
		}
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestRunKeysOnlyCached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	kind := "testRun"
	key := datastore.IncompleteKey(kind, nil)
	src := &TestDbData{TestString: "TestRunKeysOnlyCached"}
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting test data into database: %v", err)
	}

	q := datastore.NewQuery(kind).Limit(1).KeysOnly()
	for it := c.Run(ctx, q); ; {
		key, err := it.Next(nil)
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("Failed running query: %v", err)
		}
		var dst TestDbData
		c.Get(ctx, key, &dst)
	}

	q = datastore.NewQuery(kind).Limit(1).KeysOnly()
	for it := c.Run(ctx, q); ; {
		key, err := it.Next(nil)
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("Failed running query: %v", err)
		}
		var dst TestDbData
		c.Get(ctx, key, &dst)
		if dst.TestString == "" {
			t.Fatalf("Failed getting cached data. TestString was empty.")
		}
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestPutSuccess(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testPut", nil)
	src := &TestDbData{TestString: "TestPutSuccess"}

	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestPutFailInvalidSrcType(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with an invalid custom max cache size succeeded: %v", err)
	}

	key := datastore.IncompleteKey("testPut", nil)
	src := TestDbData{TestString: "TestPutFailInvalidSrcType"}
	key, err = c.Put(ctx, key, src)
	if err == nil {
		t.Fatalf("Succeeded putting invalid type into database.")

		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestPutFailInvalidCacheServer(t *testing.T) {
	ctx := context.Background()

	memcacheServers := os.Getenv("GODSCACHE_MEMCACHED_SERVERS")
	os.Setenv("GODSCACHE_MEMCACHED_SERVERS", "an invalid memcached server address")
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	os.Setenv("GODSCACHE_MEMCACHED_SERVERS", memcacheServers)
	if err != nil {
		t.Fatalf("godscache.TestPutFailInvalidCacheServer: instantiating new Client struct with an invalid custom max cache size succeeded: %v", err)
	}

	key := datastore.IncompleteKey("testPut", nil)

	src := &TestDbData{TestString: "TestPutFailInvalidCacheServer"}

	key, err = c.Put(ctx, key, src)
	if err == nil {
		t.Fatalf("godscache.TestPutFailInvalidCacheServer: Succeeded putting data into cache using invalid cache servers.")

		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("godscache.TestPutFailInvalidCacheServer: failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestPutMultiSuccess2(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("godscache.TestPutMultiSuccess2: instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	keys := make([]*datastore.Key, 0, 2)
	src := make([]*TestDbData, 0, 2)

	key := datastore.IncompleteKey("testPutMulti", nil)
	keys = append(keys, key)
	src = append(src, &TestDbData{TestString: "TestPutMultiSuccess2 1"})

	key = datastore.IncompleteKey("testPutMulti", nil)
	keys = append(keys, key)
	src = append(src, &TestDbData{TestString: "TestPutMultiSuccess2 2"})

	keys, err = c.PutMulti(ctx, keys, src)
	if err != nil {
		t.Fatalf("godscache.TestPutMultiSuccess2: failed batch putting data into database: %v", err)
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("godscache.TestPutMultiSuccess2: failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestPutMultiFail2(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("godscache.TestPutMultiSuccess2: instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	keys := make([]*datastore.Key, 0, 2)
	src := make([]*TestDbData, 0, 2)

	key := datastore.IncompleteKey("testPutMulti", nil)
	keys = append(keys, key)

	key = datastore.IncompleteKey("testPutMulti", nil)
	keys = append(keys, key)

	keys, err = c.PutMulti(ctx, keys, src)
	if err == nil {
		t.Fatalf("godscache.TestPutMultiSuccess2: succeeded batch putting data with invalid length into database.")
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("godscache.TestPutMultiSuccess2: failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestPutMultiFailInvalidCacheServers2(t *testing.T) {
	ctx := context.Background()

	memcacheServers := os.Getenv("GODSCACHE_MEMCACHED_SERVERS")
	os.Setenv("GODSCACHE_MEMCACHED_SERVERS", "an invalid memcached server address")
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	os.Setenv("GODSCACHE_MEMCACHED_SERVERS", memcacheServers)
	if err != nil {
		t.Fatalf("godscache.TestPutMultiSuccess2: instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	keys := make([]*datastore.Key, 0, 2)
	src := make([]*TestDbData, 0, 2)

	key := datastore.IncompleteKey("testPutMulti", nil)
	keys = append(keys, key)

	key = datastore.IncompleteKey("testPutMulti", nil)
	keys = append(keys, key)

	src = append(src, &TestDbData{TestString: "TestPutMultiSuccess2 1"})
	src = append(src, &TestDbData{TestString: "TestPutMultiSuccess2 2"})

	keys, err = c.PutMulti(ctx, keys, src)
	if err == nil {
		t.Fatalf("godscache.TestPutMultiSuccess2: succeeded batch putting data into cache with invalid cache servers.")
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("godscache.TestPutMultiSuccess2: failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestGetSuccessUncached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testGet", nil)
	src := &TestDbData{TestString: "TestGetSuccessUncached"}

	// Insert into database without caching.
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	var dst TestDbData
	err = c.Get(ctx, key, &dst)
	if err != nil {
		t.Fatalf("Failed getting data from database: %v", err)
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestGetSuccessCached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testGet", nil)
	src := &TestDbData{TestString: "TestGetSuccessCached"}

	// Insert into database with caching.
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	var dst TestDbData
	err = c.Get(ctx, key, &dst)
	if err != nil {
		t.Fatalf("Failed getting data from database: %v", err)
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestGetFailInvalidDstTypeUncached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testGet", nil)
	src := &TestDbData{TestString: "TestGetFailInvalidDstType"}

	// Insert into database without caching.
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	var dst TestDbData
	err = c.Get(ctx, key, dst)
	if err == nil {
		t.Fatalf("Succeeded getting data from database into an invalid dst type.")
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestGetFailInvalidDstTypeCached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testGet", nil)
	src := &TestDbData{TestString: "TestGetFailInvalidDstType"}

	// Insert into database with caching.
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	var dst TestDbData
	err = c.Get(ctx, key, dst)
	if err == nil {
		t.Fatalf("Succeeded getting data from database into an invalid dst type.")
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestGetFailUncachedInvalidCacheServers(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("godscache.TestGetFailUncachedInvalidCacheServers: instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	memcacheServers := os.Getenv("GODSCACHE_MEMCACHED_SERVERS")
	os.Setenv("GODSCACHE_MEMCACHED_SERVERS", "an invalid memcached server address")
	c2, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	os.Setenv("GODSCACHE_MEMCACHED_SERVERS", memcacheServers)
	if err != nil {
		t.Fatalf("godscache.TestGetFailUncachedInvalidCacheServers: instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testGet", nil)
	src := &TestDbData{TestString: "TestGetFailUncachedInvalidCacheServers"}

	// Insert into database with caching.
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("godscache.TestGetFailUncachedInvalidCacheServers: failed putting data into database: %v", err)
	}

	var dst TestDbData
	err = c2.Get(ctx, key, dst)
	if err == nil {
		t.Fatalf("godscache.TestGetFailUncachedInvalidCacheServers: succeeded getting data from cache using invalid cache server.")
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("godscache.TestGetFailUncachedInvalidCacheServers: failed deleting test data from datastore and cache: %v", err)
	}
}

func TestGetMultiSuccess(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	kind := "testGetMulti"
	str1 := "TestGetMultiSuccess 1"
	str2 := "TestGetMultiSuccess 2"
	str3 := "TestGetMultiSuccess 3"

	keys := make([]*datastore.Key, 0, 3)
	src := &TestDbData{TestString: str1}

	// Insert into database with caching.
	key := datastore.IncompleteKey(kind, nil)
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str2}

	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str3}

	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)

	dst := make([]*TestDbData, len(keys))

	err = c.GetMulti(ctx, keys, dst)
	if err != nil {
		t.Fatalf("Failed getting data from database: %v", err)
	}

	if dst[0].TestString == "" || dst[1].TestString == "" || dst[2].TestString == "" {
		t.Fatalf("dst is empty")
	}

	if dst[0].TestString != str1 || dst[1].TestString != str2 || dst[2].TestString != str3 {
		t.Fatalf("dst elements are in the wrong order")
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestGetMultiSuccessUncached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	kind := "testGetMulti"
	str1 := "TestGetMultiSuccessUncached 1"
	str2 := "TestGetMultiSuccessUncached 2"
	str3 := "TestGetMultiSuccessUncached 3"

	keys := make([]*datastore.Key, 0, 3)
	src := &TestDbData{TestString: str1}

	// Insert into database without caching.
	key := datastore.IncompleteKey(kind, nil)
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str2}

	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str3}

	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)

	dst := make([]*TestDbData, len(keys))

	err = c.GetMulti(ctx, keys, dst)
	if err != nil {
		t.Fatalf("Failed getting data from database: %v", err)
	}

	if dst[0].TestString == "" || dst[1].TestString == "" || dst[2].TestString == "" {
		t.Fatalf("dst is empty")
	}

	if dst[0].TestString != str1 || dst[1].TestString != str2 || dst[2].TestString != str3 {
		t.Fatalf("dst elements are in the wrong order")
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestGetMultiSuccessCachedAndUncached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	kind := "testGetMulti"
	str1 := "TestGetMultiSuccessCachedAndUncached 1"
	str2 := "TestGetMultiSuccessCachedAndUncached 2"
	str3 := "TestGetMultiSuccessCachedAndUncached 3"

	keys := make([]*datastore.Key, 0, 3)
	src := &TestDbData{TestString: str1}

	// Insert into database with caching.
	key := datastore.IncompleteKey(kind, nil)
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str2}

	// Insert into database without caching.
	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str3}

	// Insert into database with caching.
	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)

	dst := make([]*TestDbData, len(keys))

	err = c.GetMulti(ctx, keys, dst)
	if err != nil {
		t.Fatalf("Failed getting data from database: %v", err)
	}

	if dst[0].TestString == "" || dst[1].TestString == "" || dst[2].TestString == "" {
		t.Fatalf("dst is empty")
	}

	if dst[0].TestString != str1 || dst[1].TestString != str2 || dst[2].TestString != str3 {
		t.Fatalf("dst elements are in the wrong order")
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestGetMultiFail(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	kind := "testGetMulti"
	str1 := "TestGetMultiFail 1"
	str2 := "TestGetMultiFail 2"
	str3 := "TestGetMultiFail 3"

	keys := make([]*datastore.Key, 0, 3)
	src := &TestDbData{TestString: str1}

	// Insert into database with caching.
	key := datastore.IncompleteKey(kind, nil)
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str2}

	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str3}

	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)

	dst := make([]*TestDbData, len(keys))

	err = c.GetMulti(ctx, keys, &dst)
	if err == nil {
		t.Fatalf("Succeeded getting data into invalid dst type.")
	}

	err = c.GetMulti(ctx, keys, datastore.PropertyList{})
	if err == nil {
		t.Fatalf("Succeeded getting data into datastore.PropertyList, which shouldn't be allowed.")
	}

	dst = dst[:len(dst)-1]
	err = c.GetMulti(ctx, keys, dst)
	if err == nil {
		t.Fatalf("Succeeded getting data into dst of incorrect length.")
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestGetMultiFailDatastoreRequest(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	c2, err := NewClient(ctx, "a fake project")
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	kind := "testGetMulti"
	str1 := "TestGetMultiFailDatastoreRequest 1"
	str2 := "TestGetMultiFailDatastoreRequest 2"
	str3 := "TestGetMultiFailDatastoreRequest 3"

	keys := make([]*datastore.Key, 0, 3)
	src := &TestDbData{TestString: str1}

	// Insert into database with caching.
	key := datastore.IncompleteKey(kind, nil)
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str2}

	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)
	src = &TestDbData{TestString: str3}

	key = datastore.IncompleteKey(kind, nil)
	key, err = c.Parent.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	keys = append(keys, key)

	dst := make([]*TestDbData, len(keys))

	err = c2.GetMulti(ctx, keys, dst)
	if err == nil {
		t.Fatalf("Succeeded getting data from database from a fake google cloud project.")
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
		}
	}
}

func TestDeleteFailNilKey(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	err = c.Delete(ctx, nil)
	if err == nil {
		t.Fatalf("Succeeded deleting from datastore with nil key.")
	}
}

func TestDeleteFailIncompleteKey(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testDelete", nil)

	err = c.Delete(ctx, key)
	if err == nil {
		t.Fatalf("Succeeded deleting from datastore with incomplete key.")
	}
}

func TestDeleteMultiSuccess2(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	keys := make([]*datastore.Key, 0, 2)

	key := datastore.IncompleteKey("testDeleteMulti", nil)

	src := &TestDbData{TestString: "TestDeleteMultiSuccess1 1"}

	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("godscache.DeleteMultiSuccess2: failed putting data into datastore and cache: %v", err)
	}

	keys = append(keys, key)

	key = datastore.IncompleteKey("testDeleteMulti", nil)

	src = &TestDbData{TestString: "TestDeleteMultiSuccess1 2"}

	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("godscache.DeleteMultiSuccess2: failed putting data into datastore and cache: %v", err)
	}

	keys = append(keys, key)

	err = c.DeleteMulti(ctx, keys)
	if err != nil {
		t.Fatalf("godscache.DeleteMultiSuccess2: failed deleting multiple items from datastore: %v", err)
	}

	dst := make([]*TestDbData, len(keys))

	err = c.GetMulti(ctx, keys, &dst)
	if err == nil {
		t.Fatalf("godscache.DeleteMultiSuccess2: succeeded getting multiple items from datastore and cache, which were already deleted.")
	}
}

func TestDeleteMultiFail2(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	memcacheServers := os.Getenv("GODSCACHE_MEMCACHED_SERVERS")
	os.Setenv("GODSCACHE_MEMCACHED_SERVERS", "a fake server address")

	c2, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	os.Setenv("GODSCACHE_MEMCACHED_SERVERS", memcacheServers)
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	keys := make([]*datastore.Key, 0, 2)

	key := datastore.IncompleteKey("testDeleteMulti", nil)

	src := &TestDbData{TestString: "TestDeleteMultiFail1 1"}

	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("godscache.DeleteMultiFail2: failed putting data into datastore and cache: %v", err)
	}

	keys = append(keys, key)

	key = datastore.IncompleteKey("testDeleteMulti", nil)

	src = &TestDbData{TestString: "TestDeleteMultiFail1 2"}

	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("godscache.DeleteMultiFail2: failed putting data into datastore and cache: %v", err)
	}

	keys = append(keys, key)

	err = c2.DeleteMulti(ctx, keys)
	if err == nil {
		t.Fatalf("godscache.DeleteMultiFail2: succeeded deleting multiple items from datastore and cache, with invalid cache servers.")
	}

	err = c.DeleteMulti(ctx, keys)
	if err != nil {
		t.Fatalf("godscache.DeleteMultiFail2: failed deleting multiple items from datastore and cache: %v", err)
	}

	dst := make([]*TestDbData, len(keys))

	err = c.GetMulti(ctx, keys, &dst)
	if err == nil {
		t.Fatalf("godscache.DeleteMultiFail2: succeeded getting multiple items from datastore and cache, which were already deleted.")
	}
}

// ----- End Tests -----

// ----- Benchmarks -----

func BenchmarkPut(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkPut: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	keys := make([]*datastore.Key, 0)

	for i := 0; i < b.N; i++ {
		key := datastore.IncompleteKey("benchmarkPut", nil)
		key, err = c.Put(ctx, key, &TestDbData{TestString: "BenchmarkPut"})
		if err != nil {
			log.Printf("godscache.BenchmarkPut: failed putting data into datastore and cache: %v", err)
			return
		}
		keys = append(keys, key)
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.BenchmarkPut: failed deleting data from datastore and cache: %v", err)
			return
		}
	}
}

func BenchmarkPutDatastore(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkPutDatastore: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	keys := make([]*datastore.Key, 0)

	for i := 0; i < b.N; i++ {
		key := datastore.IncompleteKey("benchmarkPut", nil)
		key, err = c.Parent.Put(ctx, key, &TestDbData{TestString: "BenchmarkPutDatastore"})
		if err != nil {
			log.Printf("godscache.BenchmarkPutDatastore: failed putting data into datastore and cache: %v", err)
			return
		}
		keys = append(keys, key)
	}

	for _, key := range keys {
		err = c.Parent.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.BenchmarkPutDatastore: failed deleting data from datastore and cache: %v", err)
			return
		}
	}
}

func BenchmarkGet(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkGet: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	key := datastore.IncompleteKey("benchmarkGet", nil)
	key, err = c.Put(ctx, key, &TestDbData{TestString: "BenchmarkGet"})
	if err != nil {
		log.Printf("godscache.BenchmarkGet: failed putting data into datastore and cache: %v", err)
		return
	}

	for i := 0; i < b.N; i++ {
		var val TestDbData
		err = c.Get(ctx, key, &val)
		if err != nil {
			log.Printf("godscache.BenchmarkGet: failed getting data from datastore or cache: %v", err)
			return
		}
	}

	err = c.Delete(ctx, key)
	if err != nil {
		log.Printf("godscache.BenchmarkGet: failed deleting data from datastore and cache: %v", err)
		return
	}
}

func BenchmarkGetDatastore(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkGet: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	key := datastore.IncompleteKey("benchmarkGet", nil)
	key, err = c.Parent.Put(ctx, key, &TestDbData{TestString: "BenchmarkGetDatastore"})
	if err != nil {
		log.Printf("godscache.BenchmarkGetDatastore: failed putting data into datastore and cache: %v", err)
		return
	}

	for i := 0; i < b.N; i++ {
		var val TestDbData
		err = c.Parent.Get(ctx, key, &val)
		if err != nil {
			log.Printf("godscache.BenchmarkGetDatastore: failed getting data from datastore or cache: %v", err)
			return
		}
	}

	err = c.Parent.Delete(ctx, key)
	if err != nil {
		log.Printf("godscache.BenchmarkGetDatastore: failed deleting data from datastore and cache: %v", err)
		return
	}
}

func BenchmarkGetMulti2(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti2: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	keys := make([]*datastore.Key, 0, 2)

	key := datastore.IncompleteKey("benchmarkGetMulti", nil)
	key, err = c.Put(ctx, key, &TestDbData{TestString: "BenchmarkGetMulti2 1"})
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti2: failed putting data into datastore and cache: %v", err)
		return
	}

	keys = append(keys, key)

	key = datastore.IncompleteKey("benchmarkGetMulti", nil)
	key, err = c.Put(ctx, key, &TestDbData{TestString: "BenchmarkGetMulti2 2"})
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti2: failed putting data into datastore and cache: %v", err)
		return
	}

	keys = append(keys, key)

	vals := make([]*TestDbData, len(keys))

	for i := 0; i < b.N; i++ {
		err = c.GetMulti(ctx, keys, vals)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti2: failed getting data from datastore or cache: %v", err)
			return
		}
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti2: failed deleting data from datastore and cache: %v", err)
			return
		}
	}
}

func BenchmarkGetMulti2Datastore(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti2: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	keys := make([]*datastore.Key, 0, 2)

	key := datastore.IncompleteKey("benchmarkGetMulti", nil)
	key, err = c.Parent.Put(ctx, key, &TestDbData{TestString: "BenchmarkGetMulti2 1"})
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti2: failed putting data into datastore and cache: %v", err)
		return
	}

	keys = append(keys, key)

	key = datastore.IncompleteKey("benchmarkGetMulti", nil)
	key, err = c.Parent.Put(ctx, key, &TestDbData{TestString: "BenchmarkGetMulti2 2"})
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti2: failed putting data into datastore and cache: %v", err)
		return
	}

	keys = append(keys, key)

	vals := make([]*TestDbData, len(keys))

	for i := 0; i < b.N; i++ {
		err = c.Parent.GetMulti(ctx, keys, vals)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti2: failed getting data from datastore or cache: %v", err)
			return
		}
	}

	for _, key := range keys {
		err = c.Parent.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti2: failed deleting data from datastore and cache: %v", err)
			return
		}
	}
}

func BenchmarkGetMulti10(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti10: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	numItems := 10
	keys := make([]*datastore.Key, 0, numItems)

	for idx := 0; idx < numItems; idx++ {
		key := datastore.IncompleteKey("benchmarkGetMulti", nil)
		key, err = c.Put(ctx, key, &TestDbData{TestString: fmt.Sprintf("BenchmarkGetMulti10 %v", idx+1)})
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti10: failed putting data into datastore and cache: %v", err)
			return
		}

		keys = append(keys, key)
	}

	vals := make([]*TestDbData, len(keys))

	for i := 0; i < b.N; i++ {
		err = c.GetMulti(ctx, keys, vals)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti10: failed getting data from datastore or cache: %v", err)
			return
		}
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti10: failed deleting data from datastore and cache: %v", err)
			return
		}
	}
}

func BenchmarkGetMulti10Datastore(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti10: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	numItems := 10
	keys := make([]*datastore.Key, 0, numItems)

	for idx := 0; idx < numItems; idx++ {
		key := datastore.IncompleteKey("benchmarkGetMulti", nil)
		key, err = c.Parent.Put(ctx, key, &TestDbData{TestString: fmt.Sprintf("BenchmarkGetMulti10 %v", idx+1)})
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti10: failed putting data into datastore and cache: %v", err)
			return
		}

		keys = append(keys, key)
	}

	vals := make([]*TestDbData, len(keys))

	for i := 0; i < b.N; i++ {
		err = c.Parent.GetMulti(ctx, keys, vals)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti10: failed getting data from datastore or cache: %v", err)
			return
		}
	}

	for _, key := range keys {
		err = c.Parent.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti10: failed deleting data from datastore and cache: %v", err)
			return
		}
	}
}

func BenchmarkGetMulti100(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti100: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	numItems := 100
	keys := make([]*datastore.Key, 0, numItems)

	for idx := 0; idx < numItems; idx++ {
		key := datastore.IncompleteKey("benchmarkGetMulti", nil)
		key, err = c.Put(ctx, key, &TestDbData{TestString: fmt.Sprintf("BenchmarkGetMulti100 %v", idx+1)})
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti100: failed putting data into datastore and cache: %v", err)
			return
		}

		keys = append(keys, key)
	}

	vals := make([]*TestDbData, len(keys))

	for i := 0; i < b.N; i++ {
		err = c.GetMulti(ctx, keys, vals)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti100: failed getting data from datastore or cache: %v", err)
			return
		}
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti100: failed deleting data from datastore and cache: %v", err)
			return
		}
	}
}

func BenchmarkGetMulti100Datastore(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkGetMulti100: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	numItems := 100
	keys := make([]*datastore.Key, 0, numItems)

	for idx := 0; idx < numItems; idx++ {
		key := datastore.IncompleteKey("benchmarkGetMulti", nil)
		key, err = c.Parent.Put(ctx, key, &TestDbData{TestString: fmt.Sprintf("BenchmarkGetMulti100 %v", idx+1)})
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti100: failed putting data into datastore and cache: %v", err)
			return
		}

		keys = append(keys, key)
	}

	vals := make([]*TestDbData, len(keys))

	for i := 0; i < b.N; i++ {
		err = c.Parent.GetMulti(ctx, keys, vals)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti100: failed getting data from datastore or cache: %v", err)
			return
		}
	}

	for _, key := range keys {
		err = c.Parent.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.BenchmarkGetMulti100: failed deleting data from datastore and cache: %v", err)
			return
		}
	}
}

func BenchmarkRun1(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkRun1: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	kind := "benchmarkRun"

	key := datastore.IncompleteKey(kind, nil)
	_, err = c.Put(ctx, key, &TestDbData{TestString: "BenchmarkRun1"})
	if err != nil {
		log.Printf("godscache.BenchmarkRun1: failed putting data into datastore and cache: %v", err)
		return
	}

	for i := 0; i < b.N; i++ {
		q := datastore.NewQuery(kind).KeysOnly()

		for t := c.Run(ctx, q); ; {
			key, err := t.Next(nil)
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("godscache.BenchmarkRun1: failed getting query result: %v", err)
				return
			}

			var result TestDbData

			err = c.Get(ctx, key, &result)
			if err != nil {
				log.Printf("godscache.BenchmarkRun1: failed getting data from datastore or cache: %v", err)
				return
			}

			err = c.Delete(ctx, key)
			if err != nil {
				log.Printf("godscache.BenchmarkRun1: failed deleting data from datastore and cache: %v", err)
				return
			}
		}
	}
}

func BenchmarkRun1Datastore(b *testing.B) {
	ctx := context.Background()

	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.BenchmarkRun1Datastore: instantiating new Client struct with a valid GCP project ID failed: %v", err)
		return
	}

	kind := "benchmarkRun1Datastore"

	key := datastore.IncompleteKey(kind, nil)
	_, err = c.Parent.Put(ctx, key, &TestDbData{TestString: "BenchmarkRun1Datastore"})
	if err != nil {
		log.Printf("godscache.BenchmarkRun1Datastore: failed putting data into datastore and cache: %v", err)
		return
	}

	for i := 0; i < b.N; i++ {
		q := datastore.NewQuery(kind).KeysOnly()

		for t := c.Parent.Run(ctx, q); ; {
			key, err := t.Next(nil)
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("godscache.BenchmarkRun1Datastore: failed getting query result: %v", err)
				return
			}

			var result TestDbData

			err = c.Parent.Get(ctx, key, &result)
			if err != nil {
				log.Printf("godscache.BenchmarkRun1Datastore: failed getting data from datastore or cache: %v", err)
				return
			}

			err = c.Parent.Delete(ctx, key)
			if err != nil {
				log.Printf("godscache.BenchmarkRun1Datastore: failed deleting data from datastore and cache: %v", err)
				return
			}
		}
	}
}

// ----- End Benchmarks -----

// ----- Examples -----

func ExampleNewClient() {
	// Make a new context for running the queries.
	ctx := context.Background()

	// Provide the memcached server addresses in the context, or in the
	// GODSCACHE_MEMCACHED_SERVERS environment variable.
	//
	// memcacheServers := []string{"ip_address1:port", "ip_addressN:port"}
	// ctx = context.WithValue(ctx, MemcacheServerKey, memcacheServers)

	// Instantiate a new godscache client. You could also just supply the project ID string
	// directly here instead of calling os.Getenv("GODSCACHE_PROJECT_ID").
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.ExampleNewClient: failed creating new godscache client: %v", err)
		return
	}

	fmt.Printf("godscache.ExampleNewClient: client instantiated with %v memcache server(s).\n", len(c.MemcacheServers))

	// Output: godscache.ExampleNewClient: client instantiated with 1 memcache server(s).
}

func ExampleClient_Put() {
	// Make a new context for running the queries.
	ctx := context.Background()

	// Instantiate a new godscache client. You could also just supply the project ID string
	// directly here instead of calling os.Getenv("GODSCACHE_PROJECT_ID").
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.ExampleClient_Put: failed creating new godscache client: %v", err)
		return
	}

	// Create a new incomplete key for a given datastore kind. This key will be complete
	// and usable for queries after running Put() below.
	key := datastore.IncompleteKey("exampleClient_Put", nil)

	// Create test data to put into datastore and cache.
	val := &TestDbData{
		TestString: "ExampleClient_Put",
	}

	// Put data into the datastore and cache, and save to key the complete key received from
	// the datastore.
	key, err = c.Put(ctx, key, val)
	if err != nil {
		log.Printf("godscache.ExampleClient_Put: failed putting data into datastore and cache: %v", err)
		return
	}

	// Delete test data from datastore and cache.
	err = c.Delete(ctx, key)
	if err != nil {
		log.Printf("godscache.ExampleClient_Put: failed deleting data from datastore and cache: %v", err)
		return
	}

	fmt.Printf("godscache.ExampleClient_Put: is the datastore key incomplete?: %v\n", key.Incomplete())

	// Output: godscache.ExampleClient_Put: is the datastore key incomplete?: false
}

func ExampleClient_PutMulti() {
	// Make a new context for running the queries.
	ctx := context.Background()

	// Instantiate a new godscache client. You could also just supply the project ID string
	// directly here instead of calling os.Getenv("GODSCACHE_PROJECT_ID").
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.ExampleClient_PutMulti: failed creating new godscache client: %v", err)
		return
	}

	// Set the kind to use for the queries.
	kind := "exampleClient_PutMulti"

	// Initialize slices for keys and values, to be used with PutMulti() below.
	keys := make([]*datastore.Key, 0, 2)
	vals := make([]*TestDbData, 0, 2)

	// Create a new incomplete key for a given datastore kind.
	key := datastore.IncompleteKey(kind, nil)

	// Create test data to put into datastore and cache.
	val := &TestDbData{
		TestString: "ExampleClient_PutMulti 1",
	}

	// Add key and data to keys and vals slices.
	keys = append(keys, key)
	vals = append(vals, val)

	// Create another incomplete key for a given datastore kind.
	key = datastore.IncompleteKey(kind, nil)

	// Create test data to put into datastore and cache.
	val = &TestDbData{
		TestString: "ExampleClient_PutMulti 2",
	}

	// Add key and data to keys and vals slices.
	keys = append(keys, key)
	vals = append(vals, val)

	// Put data into the datastore and cache, and save to keys the complete keys received from
	// the datastore.
	keys, err = c.PutMulti(ctx, keys, vals)
	if err != nil {
		log.Printf("godscache.ExampleClient_PutMulti: failed putting data into datastore and cache: %v", err)
		return
	}

	// Delete test data from datastore and cache.
	err = c.DeleteMulti(ctx, keys)
	if err != nil {
		log.Printf("godscache.ExampleClient_PutMulti: failed deleting data from datastore and cache: %v", err)
		return
	}

	fmt.Printf("godscache.ExampleClient_PutMulti: is the first datastore key incomplete?: %v\n", keys[0].Incomplete())

	// Output: godscache.ExampleClient_PutMulti: is the first datastore key incomplete?: false
}

func ExampleClient_Get() {
	// Make a new context for running the queries.
	ctx := context.Background()

	// Instantiate a new godscache client. You could also just supply the project ID string
	// directly here instead of calling os.Getenv("GODSCACHE_PROJECT_ID").
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.ExampleClient_Get: failed creating new godscache client: %v", err)
		return
	}

	// Create a new incomplete key for a given datastore kind. This key will be complete
	// and usable for queries after running Put() below.
	key := datastore.IncompleteKey("exampleClient_Get", nil)

	// Create test data to put into datastore and cache.
	val := &TestDbData{
		TestString: "ExampleClient_Get",
	}

	// Put data into the datastore and cache, and save to key the complete key received from
	// the datastore.
	key, err = c.Put(ctx, key, val)
	if err != nil {
		log.Printf("godscache.ExampleClient_Get: failed putting data into datastore and cache: %v", err)
		return
	}

	// Create a variable which we will save the value to after Get() returns.
	var result TestDbData

	// Get the value from datastore or cache and save it in result.
	err = c.Get(ctx, key, &result)
	if err != nil {
		log.Printf("godscache.ExampleClient_Get: failed getting result from datastore or cache: %v", err)
		return
	}

	// Delete test data from datastore and cache.
	err = c.Delete(ctx, key)
	if err != nil {
		log.Printf("godscache.ExampleClient_Get: failed deleting data from datastore and cache: %v", err)
		return
	}

	fmt.Printf("godscache.ExampleClient_Get: result: %+v\n", result)

	// Output: godscache.ExampleClient_Get: result: {TestString:ExampleClient_Get}
}

func ExampleClient_GetMulti() {
	// Make a new context for running the queries.
	ctx := context.Background()

	// Instantiate a new godscache client. You could also just supply the project ID string
	// directly here instead of calling os.Getenv("GODSCACHE_PROJECT_ID").
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.ExampleClient_GetMulti: failed creating new godscache client: %v", err)
		return
	}

	// Allocate a slice to hold two datastore keys.
	keys := make([]*datastore.Key, 0, 2)

	// Create a new incomplete key for a given datastore kind. This key will be complete
	// and usable for queries after running Put() below.
	key := datastore.IncompleteKey("exampleClient_GetMulti", nil)

	// Create test data to put into datastore and cache.
	val := &TestDbData{
		TestString: "ExampleClient_GetMulti 1",
	}

	// Put data into the datastore and cache, and save to key the complete key received from
	// the datastore.
	key, err = c.Put(ctx, key, val)
	if err != nil {
		log.Printf("godscache.ExampleClient_GetMulti: failed putting data into datastore and cache: %v", err)
		return
	}

	// Add complete key to the keys slice, for use with GetMulti() below.
	keys = append(keys, key)

	// Create a second incomplete key for a given datastore kind. This key will be complete
	// and usable for queries after running Put() below.
	key = datastore.IncompleteKey("exampleClient_GetMulti", nil)

	// Create test data to put into datastore and cache.
	val = &TestDbData{
		TestString: "ExampleClient_GetMulti 2",
	}

	// Put data into the datastore and cache, and save to key the complete key received from
	// the datastore.
	key, err = c.Put(ctx, key, val)
	if err != nil {
		log.Printf("godscache.ExampleClient_GetMulti: failed putting data into datastore and cache: %v", err)
		return
	}

	// Add second complete key to the keys slice, for use with GetMulti() below.
	keys = append(keys, key)

	// Create a variable which we will save the results to after GetMulti() returns.
	// The results slice needs to be the same length as the keys slice.
	results := make([]*TestDbData, len(keys))

	// Get the value from datastore or cache and save it in results.
	err = c.GetMulti(ctx, keys, results)
	if err != nil {
		log.Printf("godscache.ExampleClient_GetMulti: failed getting results from datastore or cache: %v", err)
		return
	}

	// Go through all the keys, and delete them from datastore and cache.
	for _, key := range keys {
		// Delete test data from datastore and cache.
		err = c.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.ExampleClient_GetMulti: failed deleting data from datastore and cache: %v", err)
			return
		}
	}

	fmt.Printf("godscache.ExampleClient_GetMulti: result1: %+v\n", results[0])

	// Output: godscache.ExampleClient_GetMulti: result1: &{TestString:ExampleClient_GetMulti 1}
}

func ExampleClient_Delete() {
	// Make a new context for running the queries.
	ctx := context.Background()

	// Instantiate a new godscache client. You could also just supply the project ID string
	// directly here instead of calling os.Getenv("GODSCACHE_PROJECT_ID").
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.ExampleClient_Delete: failed creating new godscache client: %v", err)
		return
	}

	// Create a new incomplete key for a given datastore kind. This key will be complete
	// and usable for queries after running Put() below.
	key := datastore.IncompleteKey("exampleClient_Delete", nil)

	// Create test data to put into datastore and cache.
	val := &TestDbData{
		TestString: "ExampleClient_Delete",
	}

	// Put data into the datastore and cache, and save to key the complete key received from
	// the datastore.
	key, err = c.Put(ctx, key, val)
	if err != nil {
		log.Printf("godscache.ExampleClient_Delete: failed putting data into datastore and cache: %v", err)
		return
	}

	// Delete test data from datastore and cache.
	err = c.Delete(ctx, key)
	if err != nil {
		log.Printf("godscache.ExampleClient_Delete: failed deleting data from datastore and cache: %v", err)
		return
	}

	// Create a variable which we will save the value to after Get() returns.
	var result TestDbData

	// Try to get the deleted data from datastore or cache. This will result in an error.
	err = c.Get(ctx, key, &result)
	if err != nil {
		fmt.Printf("godscache.ExampleClient_Delete: failed getting result from datastore or cache: %v\n", err)
		return
	}

	// Output: godscache.ExampleClient_Delete: failed getting result from datastore or cache: datastore: no such entity
}

func ExampleClient_DeleteMulti() {
	// Make a new context for running the queries.
	ctx := context.Background()

	// Instantiate a new godscache client. You could also just supply the project ID string
	// directly here instead of calling os.Getenv("GODSCACHE_PROJECT_ID").
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.ExampleClient_DeleteMulti: failed creating new godscache client: %v", err)
		return
	}

	// Set the entity kind to be used for the queries below.
	kind := "exampleClient_DeleteMulti"

	// Make new slices to hold keys and values.
	keys := make([]*datastore.Key, 0, 2)
	vals := make([]*TestDbData, 0, 2)

	// Create a new incomplete key for a given datastore kind.
	key := datastore.IncompleteKey(kind, nil)

	// Create test data to put into datastore and cache.
	val := &TestDbData{
		TestString: "ExampleClient_DeleteMulti 1",
	}

	// Add key and value to keys and values slices.
	keys = append(keys, key)
	vals = append(vals, val)

	// Create another incomplete key for a given datastore kind.
	key = datastore.IncompleteKey(kind, nil)

	// Create another piece of test data to put into datastore and cache.
	val = &TestDbData{
		TestString: "ExampleClient_DeleteMulti 2",
	}

	// Add key and value to keys and values slices.
	keys = append(keys, key)
	vals = append(vals, val)

	// Put data into the datastore and cache, and save to keys the complete keys received from
	// the datastore.
	keys, err = c.PutMulti(ctx, keys, vals)
	if err != nil {
		log.Printf("godscache.ExampleClient_DeleteMulti: failed putting data into datastore and cache: %v", err)
		return
	}

	// Delete test data from datastore and cache.
	err = c.DeleteMulti(ctx, keys)
	if err != nil {
		log.Printf("godscache.ExampleClient_DeleteMulti: failed deleting data from datastore and cache: %v", err)
		return
	}

	// Create a variable which we will save the values to after GetMulti() returns.
	results := make([]*TestDbData, len(keys))

	// Try to get the deleted data from datastore or cache. This will result in an error.
	err = c.GetMulti(ctx, keys, results)
	if err != nil {
		fmt.Printf("godscache.ExampleClient_DeleteMulti: failed getting results from datastore or cache: %v\n", err)
		return
	}

	// Output: godscache.ExampleClient_DeleteMulti: failed getting results from datastore or cache: godscache.Client.GetMulti: failed getting multiple values from datastore: datastore: no such entity (and 1 other error)
}

func ExampleClient_Run() {
	// Make a new context for running the queries.
	ctx := context.Background()

	// Instantiate a new godscache client. You could also just supply the project ID string
	// directly here instead of calling os.Getenv("GODSCACHE_PROJECT_ID").
	c, err := NewClient(ctx, os.Getenv("GODSCACHE_PROJECT_ID"))
	if err != nil {
		log.Printf("godscache.ExampleClient_Run: failed creating new godscache client: %v", err)
		return
	}

	// Create a new incomplete key for a given datastore kind. This key will be complete
	// and usable for queries after running Put() below.
	key := datastore.IncompleteKey("exampleClient_Run", nil)

	// Create test data to put into datastore and cache.
	val := &TestDbData{
		TestString: "ExampleClient_Run",
	}

	// Put data into the datastore and cache, and save to key the complete key received from
	// the datastore.
	_, err = c.Put(ctx, key, val)
	if err != nil {
		log.Printf("godscache.ExampleClient_Run: failed putting data into datastore and cache: %v", err)
		return
	}

	// Make a new KeysOnly query, so we can look up the data with Get() so that it will
	// get cached.
	q := datastore.NewQuery("exampleClient_Run").KeysOnly()

	// Run the query and iterate over the results, passing nil to Next() because we only want
	// the keys.
	for t := c.Run(ctx, q); ; {
		// Get the next key.
		key, err := t.Next(nil)
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Printf("godscache.ExampleClient_Run: failed getting query result: %v", err)
			return
		}

		// Create a variable to hold the current result.
		var result TestDbData

		// Use the key to look up the result in a way which can be cached.
		err = c.Get(ctx, key, &result)
		if err != nil {
			log.Printf("godscache.ExampleClient_Run: failed getting data from datastore or cache: %v", err)
			return
		}

		fmt.Printf("godscache.ExampleClient_Run: result: %+v\n", result)

		// Delete test data from datastore and cache.
		err = c.Delete(ctx, key)
		if err != nil {
			log.Printf("godscache.ExampleClient_Run: failed deleting data from datastore and cache: %v", err)
			return
		}

		// For this example, we only care about the first result.
		break
	}

	// Output: godscache.ExampleClient_Run: result: {TestString:ExampleClient_Run}
}

// ----- End Examples -----
