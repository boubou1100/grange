// Package meta définit le stockage des métadonnées (dépôts, assets).
//
// Le MVP fournit une implémentation en mémoire. L'implémentation PostgreSQL
// (voir migrations/001_init.sql) viendra derrière la même interface.
package meta

import (
	"context"
	"errors"

	"github.com/boubou1100/grange/internal/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrExists   = errors.New("already exists")
)

type Store interface {
	CreateRepository(ctx context.Context, r model.Repository) error
	GetRepository(ctx context.Context, name string) (model.Repository, error)
	ListRepositories(ctx context.Context) ([]model.Repository, error)
	DeleteRepository(ctx context.Context, name string) error

	PutAsset(ctx context.Context, a model.Asset) error
	GetAsset(ctx context.Context, repo, path string) (model.Asset, error)
	ListAssets(ctx context.Context, repo string) ([]model.Asset, error)
	DeleteAsset(ctx context.Context, repo, path string) (model.Asset, error)
}
