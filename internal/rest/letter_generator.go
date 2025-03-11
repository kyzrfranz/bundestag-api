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
	"log/slog"
	"net/http"
	"os"
	"time"
)

type LetterRequest struct {
	Id      string   `json:"id,omitempty" bson:"_id"`
	Ids     []string `json:"ids"`
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

	pipeline := mongo.Pipeline{
		// Stage 1: Add a field 'idsCount' equal to the size of the ids array.
		{{"$addFields", bson.D{
			{"idsCount", bson.D{{"$size", "$ids"}}},
		}}},
		// Stage 2: Group by the composite key (address and ids) to get unique letter requests.
		{{"$group", bson.D{
			{"_id", bson.D{
				{"address", "$address"},
				{"ids", "$ids"},
			}},
			{"requestIdsCount", bson.D{{"$first", "$idsCount"}}},
		}}},
		// Stage 3: Aggregate over all unique letter requests.
		{{"$group", bson.D{
			{"_id", nil},
			{"uniqueRequests", bson.D{{"$sum", 1}}},
			{"totalIds", bson.D{{"$sum", "$requestIdsCount"}}},
		}}},
		// Stage 4: Project the output fields.
		{{"$project", bson.D{
			{"_id", 0},
			{"uniqueRequests", 1},
			{"totalIds", 1},
		}}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Execute the aggregation pipeline.
	cursor, err := h.collection.Aggregate(ctx, pipeline)
	if err != nil {
		h.logger.Error("Failed to aggregate", "error", err)
		http.Error(w, "Failed to aggregate", http.StatusInternalServerError)
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
