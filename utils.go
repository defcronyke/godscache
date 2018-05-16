// Copyright 2018 Jeremy Carter <Jeremy@JeremyCarter.ca>
// This file may only be used in accordance with the license in the LICENSE file in this directory.

package godscache

import (
	"context"
	"os"
	"reflect"
	"strings"
)

// CtxKeyMemcacheServers is a type for the context key "memcachedServers",
// used to specify which memcached servers to connect to.
type CtxKeyMemcacheServers string

// memcacheServers returns the memcached servers that will be used by the client.
// Set the context with a ctxKeyMemcacheServers("memcachedServers") key,
// with a value of []string{"ip_address1:port,ip_addressN:port"}, to specify which
// memcached servers to connect to. Alternately you can set the environment variable
// GODSCACHE_MEMCACHED_SERVERS="ip_address1:port,ip_addressN:port" instead to specify
// the memcached servers. The context value will take priority over the environment
// variables if both are present.
func memcacheServers(ctx context.Context) []string {
	// Check if the memcached servers are specified in the context. If so, use them.
	ctxMemcachedServers := reflect.ValueOf(ctx.Value(CtxKeyMemcacheServers("memcachedServers")))
	if ctxMemcachedServers.Kind() == reflect.Slice && ctxMemcachedServers.Len() > 0 {
		val, ok := ctxMemcachedServers.Interface().([]string)
		if ok {
			return val
		}
	}

	// If the memcached servers aren't specified in the context, get them from the
	// environment variable.
	serverStr := os.Getenv("GODSCACHE_MEMCACHED_SERVERS")
	if serverStr == "" {
		return nil
	}

	return strings.Split(serverStr, ",")
}
