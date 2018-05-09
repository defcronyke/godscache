package godscache

import (
	"os"
)

// Set the environment variable GODSCACHE_PROJECT_ID to your Google Cloud Platform project ID.
// This function can then be used to get the project ID, for use with NewClient() or elsewhere.
func ProjectID() string {
	projectID := os.Getenv("GODSCACHE_PROJECT_ID")

	if projectID == "" {
		projectID = "godscache"
	}

	return projectID
}
