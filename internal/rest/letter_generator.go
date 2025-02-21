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
	"sync"
	"time"
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
	CreationDate time.Time `json:"creation_date,omitempty"`
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
		letterRequest.CreationDate = time.Now()
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
	// Define a result type for concurrently generated PDFs.
	type pdfResult struct {
		zipFileName string
		pdfBytes    []byte
		err         error
	}

	results := make(chan pdfResult, len(letterRequest.Ids))
	var wg sync.WaitGroup

	// Launch one goroutine per ID.
	for _, id := range letterRequest.Ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			// Retrieve entry.
			entry, err := h.repo.Get(context.Background(), id)
			if err != nil {
				results <- pdfResult{err: fmt.Errorf("failed to get entry %s: %w", id, err)}
				return
			}
			h.logger.Info("processing entry", "mdb", entry)

			// Build PDF data.
			ldata := pdf.LetterData{
				SenderName: letterRequest.Address.Name,
				SenderAddress: pdf.Address{
					Street:  letterRequest.Address.Street,
					Number:  letterRequest.Address.Number,
					ZipCode: letterRequest.Address.Zip,
					City:    letterRequest.Address.City,
				},
				RecipientName: util.ShortSalutation(entry.Bio),
				RecipientAddress: pdf.Address{
					Street:  "Platz der Republik",
					Label:   "Deutscher Bundestag",
					ZipCode: 11011,
					City:    "Berlin",
					Number:  1,
				},
				Salutation: util.LongSalutation(entry.Bio),
				Party:      entry.Bio.Party,
			}

			// Generate PDF bytes.
			pdfBytes, err := pdf.Generate(ldata, "./static")
			if err != nil {
				results <- pdfResult{err: fmt.Errorf("failed to generate PDF for %s %s: %w", entry.Bio.FirstName, entry.Bio.LastName, err)}
				return
			}

			zipFileName := fmt.Sprintf("%s_%s.pdf", entry.Bio.FirstName, entry.Bio.LastName)
			results <- pdfResult{zipFileName: zipFileName, pdfBytes: pdfBytes}
		}(id)
	}

	// Wait for all goroutines to finish and close the channel.
	wg.Wait()
	close(results)

	// Check for errors and collect successful results.
	var pdfResults []pdfResult
	for res := range results {
		if res.err != nil {
			// You may choose to return immediately or collect multiple errors.
			return nil, res.err
		}
		pdfResults = append(pdfResults, res)
	}

	// Create the ZIP archive sequentially.
	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)
	for _, res := range pdfResults {
		fileInZip, err := zipWriter.Create(res.zipFileName)
		if err != nil {
			return nil, fmt.Errorf("failed to create file in zip: %s: %w", res.zipFileName, err)
		}
		if _, err := fileInZip.Write(res.pdfBytes); err != nil {
			return nil, fmt.Errorf("failed to write PDF to zip: %s: %w", res.zipFileName, err)
		}
	}
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close ZIP writer: %w", err)
	}

	return &zipBuffer, nil
}
