package rest

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	v1 "github.com/kyzrfranz/buntesdach/api/v1"
	"github.com/kyzrfranz/buntesdach/internal/util"
	"github.com/kyzrfranz/buntesdach/pkg/pdf"
	"github.com/kyzrfranz/buntesdach/pkg/resources"
	"go.mongodb.org/mongo-driver/mongo"
	"log/slog"
	"net/http"
	"os"
)

const (
	zipName = "deine_briefe"
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
}

type LetterHandler struct {
	repo       resources.Repository[v1.Politician]
	collection *mongo.Collection
	logger     *slog.Logger
}

func NewLetterHandler(repo resources.Repository[v1.Politician], collection *mongo.Collection) LetterHandler {
	return LetterHandler{
		repo:       repo,
		collection: collection,
		logger:     slog.New(slog.NewJSONHandler(os.Stdout, nil)),
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
		id, err := h.collection.InsertOne(context.Background(), letterRequest)
		if err != nil {
			h.logger.Error("Failed to queue", "error", err)
			http.Error(w, "Failed to queue", http.StatusInternalServerError)
			return
		}
		h.logger.Info("Queued", "id", id)
	} else {
		buffer, err := h.prepareDownload(letterRequest)

		if err != nil {
			h.logger.Error("Failed to download", "error", err)
			http.Error(w, "Failed to download", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", zipName))
		w.Header().Set("Content-Type", "application/zip")
		zipBytes := buffer.Bytes()
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))

		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(zipBytes); err != nil {
			http.Error(w, "Failed to write ZIP to response", http.StatusInternalServerError)
		}
	}

}

func (h *LetterHandler) prepareDownload(letterRequest LetterRequest) (*bytes.Buffer, error) {
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	for _, id := range letterRequest.Ids {
		entry, err := h.repo.Get(context.Background(), id)
		h.logger.Info("processing entry", "mdb", entry)

		if err != nil {
			return nil, err
		}
		ldata := pdf.LetterData{
			SenderName:       letterRequest.Address.Name,
			SenderAddress:    fmt.Sprintf("%s %d\n%d %s", letterRequest.Address.Street, letterRequest.Address.Number, letterRequest.Address.Zip, letterRequest.Address.City),
			RecipientName:    util.ShortSalutation(entry.Bio),
			RecipientAddress: "Deutscher Bundestag\nPlatz der Republik 1\n11011 Berlin",
			Salutation:       util.LongSalutation(entry.Bio),
			Party:            entry.Bio.Party,
		}

		pdfBytes, err := pdf.Generate(ldata, "./static")
		if err != nil {
			return nil, fmt.Errorf("failed to generate PDF for %s %s", entry.Bio.FirstName, entry.Bio.LastName)
		}

		zipFileName := fmt.Sprintf("%s.pdf", fmt.Sprintf("%s_%s", entry.Bio.FirstName, entry.Bio.LastName))
		fileInZip, err := zipWriter.Create(zipFileName)
		if err != nil {
			return nil, fmt.Errorf("failed to create file in zip: %s", zipFileName)
		}

		if _, err := fileInZip.Write(pdfBytes); err != nil {
			return nil, fmt.Errorf("failed to write PDF to zip: %s", zipFileName)
		}
	}

	// Close the ZIP writer to finalize the archive
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close ZIP writer")
	}

	return &zipBuffer, nil
}
