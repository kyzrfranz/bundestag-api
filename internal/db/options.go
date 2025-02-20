package db

type DatabaseClientOption func(client *v1MongoClient)

func WithUri(uri string) DatabaseClientOption {
	return func(c *v1MongoClient) {
		c.uri = uri
	}
}

func WithDbName(name string) DatabaseClientOption {
	return func(c *v1MongoClient) {
		c.databaseName = name
	}
}
