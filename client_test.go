// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

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

func TestNewClientValidProjectID(t *testing.T) {
	ctx := context.Background()

	_, err := NewClient(ctx, ProjectID())
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}
}

func TestNewClientProjectIDEnvVar(t *testing.T) {
	os.Setenv("DATASTORE_PROJECT_ID", ProjectID())

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

func TestNewClientFailCustomMaxCacheSize(t *testing.T) {
	os.Setenv("GODSCACHE_MAX_CACHE_SIZE", "abc")
	ctx := context.Background()

	_, err := NewClient(ctx, ProjectID())
	os.Unsetenv("GODSCACHE_MAX_CACHE_SIZE")
	if err == nil {
		t.Fatalf("Instantiating new Client struct with an invalid custom max cache size succeeded.")
	}
}

func TestRun(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	kind := "testRun"
	key := datastore.IncompleteKey(kind, nil)
	src := &TestDbData{TestString: "TestRun"}
	_, err = c.Put(ctx, key, src)
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
}

func TestRunKeysOnlyCached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	kind := "testRun"
	key := datastore.IncompleteKey(kind, nil)
	src := &TestDbData{TestString: "TestRunKeysOnlyCached"}
	_, err = c.Put(ctx, key, src)
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
}

func TestPutSuccess(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testPut", nil)
	src := &TestDbData{TestString: "TestPutSuccess"}

	key, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	// TODO: Delete test data.
}

func TestPutSuccessCustomMaxCacheSize(t *testing.T) {
	os.Setenv("GODSCACHE_MAX_CACHE_SIZE", "10")
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
	os.Unsetenv("GODSCACHE_MAX_CACHE_SIZE")
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testPut", nil)
	src := &TestDbData{TestString: "TestPutSuccessCustomMaxCacheSize"}

	_, err = c.Put(ctx, key, src)
	if err != nil {
		t.Fatalf("Failed putting data into database: %v", err)
	}

	// TODO: Delete test data.
}

func TestPutSuccessFullCache(t *testing.T) {
	os.Setenv("GODSCACHE_MAX_CACHE_SIZE", "2")
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
	os.Unsetenv("GODSCACHE_MAX_CACHE_SIZE")
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	for i := 0; i < 4; i++ {
		key := datastore.IncompleteKey("testPut", nil)
		src := &TestDbData{TestString: "TestPutSuccessFullCache"}

		_, err = c.Put(ctx, key, src)
		if err != nil {
			t.Fatalf("Failed putting data into database: %v", err)
		}
	}

	// TODO: Delete test data.
}

func TestPutFailInvalidSrcType(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
	if err != nil {
		t.Fatalf("Instantiating new Client struct with an invalid custom max cache size succeeded: %v", err)
	}

	key := datastore.IncompleteKey("testPut", nil)
	src := TestDbData{TestString: "TestPutFailInvalidSrcType"}
	_, err = c.Put(ctx, key, src)
	if err == nil {
		t.Fatalf("Succeeded putting invalid type into database.")
	}
}

func TestGetSuccessUncached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
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

	// TODO: Delete test data.
}

func TestGetSuccessCached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	key := datastore.IncompleteKey("testGet", nil)
	src := &TestDbData{TestString: "TestGetSuccessUncached"}

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

	// TODO: Delete test data.
}

func TestGetFailInvalidDstTypeUncached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
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

	// TODO: Delete test data.
}

func TestGetFailInvalidDstTypeCached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
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

	// TODO: Delete test data.
}

func TestGetFailDifferentDstTypeCached(t *testing.T) {
	ctx := context.Background()

	c, err := NewClient(ctx, ProjectID())
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

	// TODO: Delete test data.
}
