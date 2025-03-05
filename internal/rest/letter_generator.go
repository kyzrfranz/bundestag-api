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
	"html/template"
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
		h.logger.Info("Queued", "id", id, "collection", h.collection.Name())
		w.WriteHeader(http.StatusOK)

		return
	}
	//} else {
	//	buffer, err := h.prepareDownload(letterRequest)
	//
	//	if err != nil {
	//		h.logger.Error("Failed to download", "error", err)
	//		http.Error(w, "Failed to download", http.StatusInternalServerError)
	//		return
	//	}
	//	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", zipName))
	//	w.Header().Set("Content-Type", "application/zip")
	//	zipBytes := buffer.Bytes()
	//	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))
	//
	//	w.WriteHeader(http.StatusOK)
	//	if _, err := w.Write(zipBytes); err != nil {
	//		http.Error(w, "Failed to write ZIP to response", http.StatusInternalServerError)
	//	}
	//}

	http.Error(w, "Invalid action", http.StatusBadRequest)
}

func (h *LetterHandler) prepareDownload(letterRequest LetterRequest) (*bytes.Buffer, error) {
	// Define the task and result types.
	type pdfTask struct {
		id string
	}
	type pdfResult struct {
		zipFileName string
		pdfBytes    []byte
		err         error
	}

	tpl, err := template.ParseFiles(fmt.Sprintf("%s/letter.tpl.html", "./static"))
	if err != nil {
		return nil, err
	}

	// Channels to distribute tasks and collect results.
	tasks := make(chan pdfTask, len(letterRequest.Ids))
	results := make(chan pdfResult, len(letterRequest.Ids))

	// Set a limit on concurrent PDF generations.
	const numWorkers = 5
	var wg sync.WaitGroup

	// Worker function to process tasks.
	worker := func() {
		defer wg.Done()
		for task := range tasks {
			// Retrieve entry for the given ID.
			entry, err := h.repo.Get(context.Background(), task.id)
			if err != nil {
				results <- pdfResult{err: fmt.Errorf("failed to get entry %s: %w", task.id, err)}
				continue
			}
			h.logger.Info("processing entry", "mdb", entry.Bio)

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

			// Generate PDF bytes (tpl is assumed to be already loaded and passed in).
			pdfBytes, err := pdf.Generate(ldata, tpl)
			if err != nil {
				results <- pdfResult{err: fmt.Errorf("failed to generate PDF for %s %s: %w", entry.Bio.FirstName, entry.Bio.LastName, err)}
				continue
			}

			zipFileName := fmt.Sprintf("%s_%s.pdf", entry.Bio.FirstName, entry.Bio.LastName)
			results <- pdfResult{zipFileName: zipFileName, pdfBytes: pdfBytes}
		}
	}

	// Start a fixed number of workers.
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go worker()
	}

	// Enqueue tasks.
	for _, id := range letterRequest.Ids {
		tasks <- pdfTask{id: id}
	}
	close(tasks)

	// Wait for all workers to finish then close results channel.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results.
	var pdfResults []pdfResult
	for res := range results {
		if res.err != nil {
			return nil, res.err
		}
		pdfResults = append(pdfResults, res)
	}

	// Create ZIP archive containing all generated PDFs.
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
