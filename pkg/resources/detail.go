package resources

import (
	"context"
	"encoding/xml"
	myhttp "github.com/kyzrfranz/buntesdach/internal/http"
	"net/url"
)

type Entry interface {
	GetDetailUrl() *url.URL
	GetId() string
}

type EntryGetter interface {
	GetEntry(id string) (*Entry, error)
}

type detailRepo[T any] struct {
	getter EntryGetter
}

func NewDetailRepo[T any](getter EntryGetter) Repository[T] {
	return detailRepo[T]{
		getter: getter,
	}
}

func (p detailRepo[T]) List(ctx context.Context) []T {
	//TODO implement me
	panic("implement me")
}

func (p detailRepo[T]) Get(ctx context.Context, id string) (*T, error) {

	var dtg Entry

	entry, err := p.getter.GetEntry(id)
	if err != nil {
		return nil, err
	}

	dtg = *entry
	data, err := myhttp.FetchCachedUrl(dtg.GetDetailUrl(), NewFileCache("bio.json"))

	var detailType T
	if err = xml.Unmarshal(data, &detailType); err != nil {
		return nil, err
	}

	return &detailType, nil
}

func (p detailRepo[T]) Delete(ctx context.Context, id string) error {
	//TODO implement me
	panic("implement me")
}

func (p detailRepo[T]) Create(ctx context.Context, item *T) (*T, error) {
	//TODO implement me
	panic("implement me")
}

func (p detailRepo[T]) Update(ctx context.Context, oldItem *T, newItem *T) (*T, error) {
	//TODO implement me
	panic("implement me")
}

func (p detailRepo[T]) Name() string {
	//TODO implement me
	panic("implement me")
}
