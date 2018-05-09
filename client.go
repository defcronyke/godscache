package godscache

import (
	"context"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/option"
)

type Client struct {
	Parent *datastore.Client
}

func NewClient(ctx context.Context, projectID string, opts ...option.ClientOption) (*Client, error) {
	dsClient, err := datastore.NewClient(ctx, projectID, opts...)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Parent: dsClient,
	}

	return c, nil
}

func (c *Client) Run(ctx context.Context, q *datastore.Query) *Iterator {
	cached := false

	// TODO: Look up query in cache. If found, set cached to true.

	t := &Iterator{
		Cached: cached,
	}

	if !cached {
		t.Parent = c.Parent.Run(ctx, q)
	}

	return t
}
