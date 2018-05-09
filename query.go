package godscache

import (
	"cloud.google.com/go/datastore"
)

type Iterator struct {
	Parent *datastore.Iterator
	Cached bool
}

func (t *Iterator) Next(dst interface{}) (*Key, error) {
	var kParent *datastore.Key
	var err error

	k := &Key{
		Cached: t.Cached,
	}

	// TODO: Get from cache if t.Cached is true.
	if t.Cached {

	} else {
		kParent, err = t.Parent.Next(dst)
		k.Parent = kParent
	}

	return k, err
}
