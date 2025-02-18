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
	"log/slog"
	"net/http"
	"os"
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
	repo resources.Repository[v1.Politician]
}

func NewLetterHandler(repo resources.Repository[v1.Politician]) LetterHandler {
	return LetterHandler{
		repo: repo,
	}
}

func (h *LetterHandler) Generate(w http.ResponseWriter, req *http.Request) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var zipBuffer bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuffer)

	var letterRequest LetterRequest
	if err := json.NewDecoder(req.Body).Decode(&letterRequest); err != nil {
		logger.Error("Failed to marshal", "error", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	for _, id := range letterRequest.Ids {
		entry, err := h.repo.Get(context.Background(), id)
		logger.Info("processing entry", "mdb", entry)

		if err != nil {
			logger.Error("Failed to write pdf to response", "error", err)
			w.WriteHeader(http.StatusNotFound)
			return
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
			logger.Error("Failed to generate PDF", "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		zipFileName := fmt.Sprintf("%s.pdf", fmt.Sprintf("%s_%s", entry.Bio.FirstName, entry.Bio.LastName))
		fileInZip, err := zipWriter.Create(zipFileName)
		if err != nil {
			logger.Error("Failed to create file in zip", "file", zipFileName, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := fileInZip.Write(pdfBytes); err != nil {
			logger.Error("Failed to write PDF to zip", "file", zipFileName, "error", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Close the ZIP writer to finalize the archive
	if err := zipWriter.Close(); err != nil {
		logger.Error("Failed to close zip writer", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Now return the ZIP as a download
	w.Header().Set("Content-Disposition", "attachment; filename=letters.zip")
	w.Header().Set("Content-Type", "application/zip")
	zipBytes := zipBuffer.Bytes()
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))

	if _, err := w.Write(zipBytes); err != nil {
		logger.Error("Failed to write zip to response", "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error writing ZIP to response"))
	}
}
