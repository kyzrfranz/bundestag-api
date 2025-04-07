package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kyzrfranz/bundestag-api/internal/img"
	"github.com/kyzrfranz/bundestag-api/pkg/resources"
	"io"
	"net/http"
	"os"
)

type Link struct {
	Link string `json:"link"`
	Rel  string `json:"rel"`
}

type Handler[T any] interface {
	List(w http.ResponseWriter, r *http.Request)
	Get(w http.ResponseWriter, r *http.Request)
	Create(w http.ResponseWriter, r *http.Request)
	Update(w http.ResponseWriter, r *http.Request)
	Delete(w http.ResponseWriter, r *http.Request)
	Path() string
}

type genericHandler[T any] struct {
	repo    resources.Repository[T]
	context context.Context
}

func NewHandler[T any](resourceRepo resources.Repository[T]) Handler[T] {
	return genericHandler[T]{
		repo: resourceRepo,
	}
}

func (r genericHandler[T]) List(w http.ResponseWriter, req *http.Request) {
	res := r.repo.List(r.context)

	if err := MarshalResponse(w, res); err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
}

func (r genericHandler[T]) Get(w http.ResponseWriter, req *http.Request) {
	id := req.PathValue("id")
	res, err := r.repo.Get(r.context, id)

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if req.Header.Get("Accept") == "image/webp" {
		err := img.EnsureImage(res, id)
		if err != nil {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		f, err := os.Open(fmt.Sprintf(".img/%s.webp", id))
		if err != nil {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		defer f.Close()

		w.Header().Set("Content-Type", "image/webp")
		w.WriteHeader(http.StatusOK)

		if _, err := io.Copy(w, f); err != nil {
			http.Error(w, "Failed to stream image", http.StatusInternalServerError)
			return
		}
	} else {
		if err = MarshalResponse(w, res); err != nil {
			http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
			return
		}
	}

}

func (r genericHandler[T]) Create(w http.ResponseWriter, req *http.Request) {

}

func (r genericHandler[T]) Update(w http.ResponseWriter, req *http.Request) {

}

func (r genericHandler[T]) Delete(w http.ResponseWriter, req *http.Request) {

}

func (r genericHandler[T]) Path() string {
	return fmt.Sprintf("/%s", r.repo.Name())
}

func MarshalResponse(w http.ResponseWriter, res interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	jsonData, err := json.Marshal(res)
	if err != nil {
		return err
	}
	_, writeErr := w.Write(jsonData) // Write the JSON data
	if writeErr != nil {
		return writeErr
	}
	return nil
}
