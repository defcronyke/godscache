// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

package godscache

import (
	"os"
	"strings"
)

// ProjectID is used by the package tests. Before you run the tests, you have to set
// the GODSCACHE_PROJECT_ID environment variable to a Google Cloud Platform project ID
// of a project you control that has an initialized Datastore.
func ProjectID() string {
	projectID := os.Getenv("GODSCACHE_PROJECT_ID")

	if projectID == "" {
		projectID = "godscache"
	}

	return projectID
}

// MemcacheServers returns the memcached servers that will be used by the client.
// Set the environment variable GODSCACHE_MEMCACHED_SERVERS="ip_address1:port,ip_addressN:port"
// to specify which memcached servers to connect to.
func MemcacheServers() []string {
	serverStr := os.Getenv("GODSCACHE_MEMCACHED_SERVERS")
	if serverStr == "" {
		return []string{
			"35.203.95.85:11211",
			"35.203.77.98:11211",
		}
	}

	return strings.Split(serverStr, ",")
}
