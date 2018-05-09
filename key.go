package godscache

import "cloud.google.com/go/datastore"

type Key struct {
	Parent *datastore.Key
	Cached bool
}
