package rest

import (
	"context"
	"encoding/json"
	v1 "github.com/kyzrfranz/buntesdach/api/v1"
	"github.com/kyzrfranz/buntesdach/pkg/resources"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log/slog"
	"net/http"
	"os"
	"time"
)

type LetterRequest struct {
	Ids     []string `json:"ids"`
	Address struct {
		Name   string `json:"name"`
		Street string `json:"street"`
		Number int    `json:"number"`
		Zip    int    `json:"zip"`
		City   string `json:"city"`
	} `json:"address"`
	CreationDate time.Time `json:"creation_date,omitempty"`
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
