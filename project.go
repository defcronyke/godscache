// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

package godscache

import (
	"os"
)

// This function is used by the package tests. Before you run the tests, you have to set
// the GODSCACHE_PROJECT_ID environment variable to a Google Cloud Platform project ID
// of a project you control that has an initialized Datastore.
func projectID() string {
	projectID := os.Getenv("GODSCACHE_PROJECT_ID")

	if projectID == "" {
		projectID = "godscache"
	}

	return projectID
}
