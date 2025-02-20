package db

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"net/http"
)

type DatabaseClient[Item any] interface {
	List(ctx context.Context, filter bson.M) []Item
	Get(ctx context.Context, id string) (*Item, error)
	Create(ctx context.Context, item *Item) (interface{}, error)
	Update(ctx context.Context, id string, update bson.M) (interface{}, error)
	Delete(ctx context.Context, id string) (int64, error)
}

type MongoCLient struct {
	httpClient     *http.Client
	baseURL        string
	dbName         string
	collectionName string
	username       string
	password       string
	baseUrl        string
}
