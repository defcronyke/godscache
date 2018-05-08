package godscache

import (
	"context"
	"log"

	"cloud.google.com/go/datastore"
)

type godscache struct {
	gcpProjectID string
	ctx          context.Context
	dsClient     *datastore.Client
}

func NewGodscache(gcpProjectID string) *godscache {
	ctx := context.Background()

	dsClient, err := datastore.NewClient(ctx, gcpProjectID)
	if err != nil {
		log.Printf("Error: Failed creating new datastore client: %v", err)
		return nil
	}

	g := &godscache{
		gcpProjectID: gcpProjectID,
		ctx:          ctx,
		dsClient:     dsClient,
	}

	log.Printf("Instantiated new Godscache: %+v", g)
	return g
}
