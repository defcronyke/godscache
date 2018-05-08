package godscache

import (
	"os"
	"testing"
)

// Things that should succeed.
func TestNewGodscacheWithValidProjectID(t *testing.T) {
	g := NewGodscache("godscache")

	if g == nil {
		t.Fatalf("Instantiating new Godscache struct with a valid GCP project ID failed")
	}
}

func TestNewGodscacheWithProjectIDFromEnvVar(t *testing.T) {
	os.Setenv("DATASTORE_PROJECT_ID", "godscache")

	g := NewGodscache("")

	os.Unsetenv("DATASTORE_PROJECT_ID")

	if g == nil {
		t.Fatalf("Instantiating new Godscache struct with project ID in the DATASTORE_PROJECT_ID environment variable failed")
	}
}

// Things that should fail.
func TestNewGodscacheWithNoProjectID(t *testing.T) {
	g := NewGodscache("")

	if g != nil {
		t.Fatalf("Instantiating new Godscache struct with no project ID succeeded")
	}
}
