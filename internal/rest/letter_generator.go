package rest

import (
	"context"
	"encoding/json"
	v1 "github.com/kyzrfranz/buntesdach/api/v1"
	"github.com/kyzrfranz/buntesdach/pkg/resources"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	StatusQueued = "queued"
	StatusSent   = "sent"
)

type LetterRequest struct {
	Id      string   `json:"id,omitempty" bson:"_id"`
	Ids     []string `json:"ids"`
	MyMdbs  []string `json:"myMdbs,omitempty"`
	Status  string   `json:"status,omitempty"`
	Address struct {
		Name   string `json:"name"`
		Street string `json:"street"`
		Number int    `json:"number"`
		Zip    string `json:"zip"`
		City   string `json:"city"`
	} `json:"address"`
	CreationDate time.Time `json:"creation_date,omitempty"`
}

type Stats struct {
	UniqueRequests int `bson:"uniqueRequests" json:"uniqueRequests"`
	TotalIds       int `bson:"totalIds" json:"totalLetters"`
	TopIds         []struct {
		ID    string `bson:"_id" json:"id"`
		Count int    `bson:"count" json:"count"`
	} `bson:"topIds" json:"topIds"`
	StatusCounts struct {
		Queued int `bson:"queued" json:"queued"`
		Sent   int `bson:"sent" json:"sent"`
	} `bson:"statusCounts"`
}

type LetterHandler struct {
	repo       resources.Repository[v1.Politician]
	collection *mongo.Collection
	logger     *slog.Logger
	authKey    string
}

func NewLetterHandler(repo resources.Repository[v1.Politician], collection *mongo.Collection, key string) LetterHandler {
	return LetterHandler{
		repo:       repo,
		collection: collection,
		logger:     slog.New(slog.NewJSONHandler(os.Stdout, nil)),
		authKey:    key,
	}
}

func (h *LetterHandler) Handle(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodPost:
		h.Generate(w, req)
	case http.MethodGet:
		h.List(w, req)
	case http.MethodPatch:
		h.Patch(w, req)
	case http.MethodDelete:
		h.Delete(w, req)
	default:
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}

func (h *LetterHandler) Generate(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var letterRequest LetterRequest
	if err := json.NewDecoder(req.Body).Decode(&letterRequest); err != nil {
		h.logger.Error("Failed to marshal", "error", err)
		http.Error(w, "request object is invalid ", http.StatusBadRequest)
		return
	}

	actionParam := req.URL.Query().Get("action")
	if actionParam == "queue" {
		letterRequest.Status = StatusQueued
		letterRequest.CreationDate = time.Now()
		letterRequest.Id = primitive.NewObjectID().Hex()
		id, err := h.collection.InsertOne(context.Background(), letterRequest)
		if err != nil {
			h.logger.Error("Failed to queue", "error", err)
			http.Error(w, "Failed to queue", http.StatusInternalServerError)
			return
		}
		h.logger.Info("Queued", "id", id, "collection", h.collection.Name())
		w.WriteHeader(http.StatusOK)

		return
	}
	http.Error(w, "Invalid action", http.StatusBadRequest)
}

func (h *LetterHandler) Patch(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPatch {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Authorization") != "Bearer "+h.authKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	id := req.PathValue("id")
	if id == "" {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	var patchRequest map[string]interface{}
	if err := json.NewDecoder(req.Body).Decode(&patchRequest); err != nil {
		h.logger.Error("Failed to decode", "error", err)
		http.Error(w, "Failed to decode", http.StatusInternalServerError)
		return
	}

	update := bson.M{"$set": patchRequest}
	res, err := h.collection.UpdateOne(context.Background(),
		bson.M{"_id": id}, update)
	if err != nil {
		objectId, err := primitive.ObjectIDFromHex(id)
		res, err = h.collection.UpdateOne(context.Background(),
			bson.M{"_id": objectId}, update)
		if err != nil {
			h.logger.Error("Failed to update", "error", err)
			http.Error(w, "Failed to update", http.StatusInternalServerError)
			return
		}
	}
	if res.MatchedCount == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *LetterHandler) Delete(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if req.Header.Get("Authorization") != "Bearer "+h.authKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	id := req.PathValue("id")
	if id == "" {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}

	//objectId, err := primitive.ObjectIDFromHex(id) - activate for old data

	res, err := h.collection.DeleteOne(context.Background(), bson.M{"_id": id})
	if res.DeletedCount == 0 {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.logger.Error("Failed to delete", "error", err)
		http.Error(w, "Failed to delete", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *LetterHandler) Stats(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	top := req.URL.Query().Get("top")
	topIds, err := strconv.Atoi(top)
	if err != nil {
		topIds = 10
	}

	pipeline := mongo.Pipeline{
		{{"$facet", bson.D{
			// Facet for unique summary based on address & ids
			{"uniqueSummary", bson.A{
				bson.D{{"$addFields", bson.D{
					{"idsCount", bson.D{{"$size", "$ids"}}},
				}}},
				bson.D{{"$group", bson.D{
					{"_id", bson.D{
						{"address", "$address"},
						{"ids", "$ids"},
					}},
					{"requestIdsCount", bson.D{{"$first", "$idsCount"}}},
				}}},
				bson.D{{"$group", bson.D{
					{"_id", nil},
					{"uniqueRequests", bson.D{{"$sum", 1}}},
					{"totalIds", bson.D{{"$sum", "$requestIdsCount"}}},
				}}},
				bson.D{{"$project", bson.D{
					{"_id", 0},
					{"uniqueRequests", 1},
					{"totalIds", 1},
				}}},
			}},
			{"topIds", bson.A{
				bson.D{{"$unwind", "$ids"}},
				bson.D{{"$group", bson.D{
					{"_id", "$ids"},
					{"count", bson.D{{"$sum", 1}}},
				}}},
				bson.D{{"$sort", bson.D{{"count", -1}}}},
				bson.D{{"$limit", topIds}},
			}},
			// Facet for status counts with deduplication.
			{"statusCounts", bson.A{
				// First group by composite key to remove duplicates.
				bson.D{{"$group", bson.D{
					{"_id", bson.D{
						{"address", "$address"},
						{"ids", "$ids"},
					}},
					{"status", bson.D{{"$first", "$status"}}},
				}}},
				// Now group all unique records and count statuses.
				bson.D{{"$group", bson.D{
					{"_id", nil},
					{"queued", bson.D{{"$sum", bson.D{{"$cond", bson.A{
						bson.D{{"$eq", bson.A{"$status", "queued"}}},
						1,
						0,
					}}}}}},
					{"sent", bson.D{{"$sum", bson.D{{"$cond", bson.A{
						bson.D{{"$eq", bson.A{"$status", "sent"}}},
						1,
						0,
					}}}}}},
				}}},
				bson.D{{"$project", bson.D{
					{"_id", 0},
					{"queued", 1},
					{"sent", 1},
				}}},
			}},
		}}},
		{{"$project", bson.D{
			{"uniqueRequests", bson.D{{"$arrayElemAt", bson.A{"$uniqueSummary.uniqueRequests", 0}}}},
			{"totalIds", bson.D{{"$arrayElemAt", bson.A{"$uniqueSummary.totalIds", 0}}}},
			{"topIds", 1},
			{"statusCounts", bson.D{{"$arrayElemAt", bson.A{"$statusCounts", 0}}}},
		}}},
	}

	// Set a timeout for the aggregation operation.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cursor, err := h.collection.Aggregate(ctx, pipeline)
	if err != nil {
		log.Fatal("Aggregation error: ", err)
	}
	defer cursor.Close(ctx)

	var results []Stats
	if err = cursor.All(ctx, &results); err != nil {
		h.logger.Error("Failed to decode", "error", err)
		http.Error(w, "Failed to decode", http.StatusInternalServerError)
	}

	if err := marshalResponse(w, results); err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
}

func (h *LetterHandler) List(w http.ResponseWriter, req *http.Request) {

	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if h.authKey == "" {
		h.logger.Error("Auth key is empty")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	//check Authorization
	if req.Header.Get("Authorization") != "Bearer "+h.authKey {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	var letters []LetterRequest
	opts := options.Find().SetSort(bson.D{{"created_date", -1}})
	cursor, err := h.collection.Find(context.Background(), bson.M{}, opts)
	if err != nil {
		h.logger.Error("Failed to list", "error", err)
		http.Error(w, "Failed to list", http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.Background())
	for cursor.Next(context.Background()) {
		var letter LetterRequest
		if err := cursor.Decode(&letter); err != nil {
			h.logger.Error("Failed to decode", "error", err)
			http.Error(w, "Failed to decode", http.StatusInternalServerError)
			return
		}
		letters = append(letters, letter)
	}
	if err := cursor.Err(); err != nil {
		h.logger.Error("Failed to list", "error", err)
		http.Error(w, "Failed to list", http.StatusInternalServerError)
		return
	}

	if err := marshalResponse(w, letters); err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}

}
