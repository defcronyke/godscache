// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

package godscache

import (
	"os"
)

// Set the environment variable GODSCACHE_PROJECT_ID to your Google Cloud Platform project ID.
// This function can then be used to get the project ID, for use with NewClient() or elsewhere.
// If you want to run the package tests, it is required to set the environment variable to a
// valid GCP project ID of a project that you control, with an initialized datastore.
func ProjectID() string {
	projectID := os.Getenv("GODSCACHE_PROJECT_ID")

	if projectID == "" {
		projectID = "godscache"
	}

	return projectID
}
