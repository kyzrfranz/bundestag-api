package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kyzrfranz/buntesdach/pkg/resources"
	"net/http"
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

	if err := marshalResponse(w, res); err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
	}
}

func (r genericHandler[T]) Get(w http.ResponseWriter, req *http.Request) {
	res, err := r.repo.Get(r.context, req.PathValue("id"))

	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if err = marshalResponse(w, res); err != nil {
		http.Error(w, "Failed to marshal response", http.StatusInternalServerError)
		return
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

func marshalResponse(w http.ResponseWriter, res interface{}) error {
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
