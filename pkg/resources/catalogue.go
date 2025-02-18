package resources

import (
	"context"
	"fmt"
)

type catalogueRepo[E any] struct {
	getter CatalogueDataGetter[E]
}

type CatalogueDataGetter[E any] interface {
	CatalogueGetter[E]
	CatalogueEntryGetter[E]
}

type CatalogueGetter[E any] interface {
	GetCatalog() ([]E, error)
}

type CatalogueEntryGetter[E any] interface {
	GetCatalogueEntry(id string) (*E, error)
}

func NewCatalogueRepo[E any](getter CatalogueDataGetter[E]) Repository[E] {
	return catalogueRepo[E]{
		getter: getter,
	}
}

func (s catalogueRepo[E]) List(ctx context.Context) []E {
	catalog, err := s.getter.GetCatalog()
	if err != nil {
		fmt.Printf("failed to get catalogue: %v\n", err)
		return nil
	}
	return catalog
}

func (s catalogueRepo[E]) Get(ctx context.Context, id string) (*E, error) {
	return s.getter.GetCatalogueEntry(id)
}

func (s catalogueRepo[E]) Delete(ctx context.Context, id string) error {
	//TODO implement me
	panic("implement me")
}

func (s catalogueRepo[E]) Create(ctx context.Context, item *E) (*E, error) {
	//TODO implement me
	panic("implement me")
}

func (s catalogueRepo[E]) Update(ctx context.Context, oldItem *E, newItem *E) (*E, error) {
	//TODO implement me
	panic("implement me")
}

func (s catalogueRepo[E]) Name() string {
	//TODO implement me
	panic("implement me")
}
