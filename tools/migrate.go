package main

import (
	"context"
	"fmt"
	"github.com/kyzrfranz/buntesdach/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"log"
	"log/slog"
	"os"
)

var (
	logger          *slog.Logger
	mongoUri        string
	mongoCollection string
)

func main() {
	mongoUri = stringOrEnv("MONGO_URI", "")
	mongoCollection = stringOrEnv("MONGO_COLLECTION", "test")

	cli, err := db.NewV1MongoClient(db.WithUri(mongoUri))
	if err != nil {
		logger.Error("failed to connect to mongo", "error", err)
		os.Exit(1)
	}
	collection := cli.Database("buntesdach").Collection(mongoCollection)

	defer cli.Disconnect(context.Background())

	// Filter documents where address.zip is stored as an int.
	filter := bson.M{
		"address.zip": bson.M{"$type": "int"},
	}

	// Define an aggregation pipeline update to convert address.zip to a string.
	updatePipeline := mongo.Pipeline{
		{{"$set", bson.D{
			{"address.zip", bson.D{{"$toString", "$address.zip"}}},
		}}},
	}

	updateResult, err := collection.UpdateMany(context.Background(), filter, updatePipeline)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Matched %d documents and updated %d documents\n", updateResult.MatchedCount, updateResult.ModifiedCount)
}

func stringOrEnv(key string, defaultVal string) (s string) {
	s = os.Getenv(key)
	if s != "" {
		defaultVal = s
	}

	return defaultVal
}
