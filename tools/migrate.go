package main

import (
	"context"
	"fmt"
	"github.com/kyzrfranz/buntesdach/internal/db"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	migrateIdObjectIdToString(collection)

	//fmt.Printf("Matched %d documents and updated %d documents\n", updateResult.MatchedCount, updateResult.ModifiedCount)
}

func migrateIdObjectIdToString(collection *mongo.Collection) {
	ctx := context.Background()

	// Find all documents where _id is an ObjectId
	cursor, err := collection.Find(ctx, bson.M{"_id": bson.M{"$type": "objectId"}})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(ctx)

	var docs []bson.M
	if err := cursor.All(ctx, &docs); err != nil {
		log.Fatal(err)
	}

	for _, doc := range docs {

		//Convert ObjectId to string
		objectID, ok := doc["_id"].(primitive.ObjectID)
		if !ok {
			continue
		}
		newID := objectID.Hex() // Convert ObjectId to string

		// Replace _id with new string-based ID
		doc["_id"] = newID

		// Insert new document with string _id
		_, err := collection.InsertOne(ctx, doc)
		if err != nil {
			log.Printf("Failed to insert new document for _id %s: %v\n", newID, err)
			continue
		}

		// Delete the old document with ObjectId
		_, err = collection.DeleteOne(ctx, bson.M{"_id": objectID})
		if err != nil {
			log.Printf("Failed to delete old document for _id %s: %v\n", objectID.Hex(), err)
		}
	}

	fmt.Println("Migration completed.")
}

func migrateZipcode(collection *mongo.Collection) *mongo.UpdateResult {
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

	return updateResult
}

func stringOrEnv(key string, defaultVal string) (s string) {
	s = os.Getenv(key)
	if s != "" {
		defaultVal = s
	}

	return defaultVal
}
