package godscache

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/iterator"
)

const PROJECT_ID = "godscache" // NOTE: Change this to your GCP project ID.

type EmptyKind struct{}

// Things that should succeed.
func TestNewClientWithValidProjectID(t *testing.T) {
	ctx := context.Background()
	_, err := NewClient(ctx, PROJECT_ID)
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}
}

func TestNewClientWithProjectIDFromEnvVar(t *testing.T) {
	os.Setenv("DATASTORE_PROJECT_ID", PROJECT_ID)

	ctx := context.Background()
	_, err := NewClient(ctx, "")
	if err != nil {
		t.Fatalf("Instantiating new Client struct with project ID in the DATASTORE_PROJECT_ID environment variable failed: %v", err)
	}

	os.Unsetenv("DATASTORE_PROJECT_ID")
}

func TestRunNoResults(t *testing.T) {
	ctx := context.Background()
	c, err := NewClient(ctx, PROJECT_ID)
	if err != nil {
		t.Fatalf("Instantiating new Client struct with a valid GCP project ID failed: %v", err)
	}

	q := datastore.NewQuery("notARealKind").Limit(1)
	var it *Iterator
	for it = c.Run(ctx, q); ; {
		var res EmptyKind
		_, err := it.Next(&res)
		if err != iterator.Done {
			t.Fatalf("Got results when testing c.Run: %v", err)
		}
		break
	}
}

// Things that should fail.
func TestNewClientWithNoProjectID(t *testing.T) {
	ctx := context.Background()
	_, err := NewClient(ctx, "")
	if err == nil {
		t.Fatalf("Instantiating new Client struct with no project ID succeeded: %v", err)
	}
}
