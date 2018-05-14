// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

package godscache

import (
	"os"
	"strings"
)

// MemcacheServers returns the memcached servers that will be used by the client.
// Set the environment variable GODSCACHE_MEMCACHED_SERVERS="ip_address1:port,ip_addressN:port"
// to specify which memcached servers to connect to.
func MemcacheServers() []string {
	serverStr := os.Getenv("GODSCACHE_MEMCACHED_SERVERS")
	if serverStr == "" {
		return nil
	}

	return strings.Split(serverStr, ",")
}
