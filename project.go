package godscache

import (
	"os"
)

// Set the envirnment variable GODSCACHE_PROJECT_ID to your Google Cloud Platform project ID.
func ProjectID() string {
	projectID := os.Getenv("GODSCACHE_PROJECT_ID")

	if projectID == "" {
		projectID = "godscache"
	}

	return projectID
}
