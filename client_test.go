// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

// Set the environment variable GODSCACHE_PROJECT_ID to your Google Cloud Platform project ID before running these tests.
// It must be set to a valid GCP project ID of a project that you control, with an initialized datastore.

package godscache

import (
	"context"
	"log"
	"os"
	"testing"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

type EmptyKind struct{}

type TestDbData struct {
	TestString string
}

type TestDbDataDifferent struct {
	TestString string
}

func TestNewClientValidprojectID(t *testing.T) {
	ctx := context.Background()

	_, err := NewClient(ctx, projectID())
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}
}

func TestNewClientProjectIDEnvVar(t *testing.T) {
	os.Setenv("DATASTORE_PROJECT_ID", projectID())

	ctx := context.Background()
	_, err := NewClient(ctx, "")
	if err != nil {
		t.Fatalf("Instantiating new Client struct with project ID in the DATASTORE_projectID environment variable failed: %v", err)
	}

	os.Unsetenv("DATASTORE_PROJECT_ID")
}

func TestNewClientNoprojectID(t *testing.T) {
	ctx := context.Background()

	_, err := NewClient(ctx, "")
	if err == nil {
		t.Fatalf("Instantiating new Client struct with no project ID succeeded.")
	}
}

func TestNewClientFailCustomMaxCacheSize(t *testing.T) {
	os.Setenv("GODSCACHE_MAX_CACHE_SIZE", "abc")
	ctx := context.Background()

	_, err := NewClient(ctx, projectID())
	os.Unsetenv("GODSCACHE_MAX_CACHE_SIZE")
	if err == nil {
		t.Fatalf("Instantiating new Client struct with an invalid custom max cache size succeeded.")
	}
}

func TestRun(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, projectID())
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
		log.Printf("Received test data: %+v", res)
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestRunKeysOnlyCached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, projectID())
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
		log.Printf("Got test data: %+v", dst)
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
		log.Printf("Got test data: %+v", dst)
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

	c, err := NewClient(ctx, projectID())
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

func TestPutSuccessCustomMaxCacheSize(t *testing.T) {
	os.Setenv("GODSCACHE_MAX_CACHE_SIZE", "10")
	ctx := context.Background()

	c, err := NewClient(ctx, projectID())
	os.Unsetenv("GODSCACHE_MAX_CACHE_SIZE")
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testPut", nil)
	src := &TestDbData{TestString: "TestPutSuccessCustomMaxCacheSize"}

	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestPutSuccessFullCache(t *testing.T) {
	os.Setenv("GODSCACHE_MAX_CACHE_SIZE", "2")
	ctx := context.Background()

	c, err := NewClient(ctx, projectID())
	os.Unsetenv("GODSCACHE_MAX_CACHE_SIZE")
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	keys := make([]*datastore.Key, 0, c.MaxCacheSize)

	for i := 0; i < 4; i++ {
		key := datastore.IncompleteKey("testPut", nil)
		src := &TestDbData{TestString: "TestPutSuccessFullCache"}

		key, err = c.Put(ctx, key, src)
		if err != nil {
			t.Fatalf("Failed putting data into database: %v", err)
		}

		keys = append(keys, key)
	}

	for _, key := range keys {
		err = c.Delete(ctx, key)
		if err != nil {
			t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
		}
	}

	keys = nil
}

func TestPutFailInvalidSrcType(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, projectID())
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

func TestGetSuccessUncached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, projectID())
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

	c, err := NewClient(ctx, projectID())
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

	c, err := NewClient(ctx, projectID())
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

	c, err := NewClient(ctx, projectID())
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

func TestGetFailDifferentDstTypeCached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, projectID())
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

	var dst TestDbDataDifferent
	err = c.Get(ctx, key, &dst)
	if err == nil {
		t.Fatalf("Succeeded getting data from database into a different dst type.")
	}

	err = c.Delete(ctx, key)
	if err != nil {
		t.Fatalf("Failed deleting test data from datastore and cache: %v", err)
	}
}

func TestGetMultiSuccess(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, projectID())
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

	c, err := NewClient(ctx, projectID())
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

	log.Printf("dst: %+v", dst)

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

	c, err := NewClient(ctx, projectID())
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

	log.Printf("dst: %+v", dst)

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

	c, err := NewClient(ctx, projectID())
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

	c, err := NewClient(ctx, projectID())
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	c2, err := NewClient(ctx, "a fake project")
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

	err = c2.GetMulti(ctx, keys, dst)
	if err == nil {
		t.Fatalf("Succeeded getting data from database from a fake google cloud project.")
	}
}

func TestDeleteFailIncompleteKey(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, projectID())
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testDelete", nil)

	err = c.Delete(ctx, key)
	if err == nil {
		t.Fatalf("Succeeded deleting from datastore with incomplete key.")
	}
}
