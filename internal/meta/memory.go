package meta

import (
	"context"
	"sort"
	"sync"

	"github.com/boubou1100/grange/internal/model"
)

// Memory est une implémentation en mémoire (dev et tests uniquement).
type Memory struct {
	mu     sync.RWMutex
	repos  map[string]model.Repository
	assets map[string]model.Asset // clé: repo + "\x00" + path
}

func NewMemory() *Memory {
	return &Memory{repos: map[string]model.Repository{}, assets: map[string]model.Asset{}}
}

func key(repo, path string) string { return repo + "\x00" + path }

func (m *Memory) CreateRepository(_ context.Context, r model.Repository) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.repos[r.Name]; ok {
		return ErrExists
	}
	m.repos[r.Name] = r
	return nil
}

func (m *Memory) GetRepository(_ context.Context, name string) (model.Repository, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.repos[name]
	if !ok {
		return model.Repository{}, ErrNotFound
	}
	return r, nil
}

func (m *Memory) ListRepositories(_ context.Context) ([]model.Repository, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]model.Repository, 0, len(m.repos))
	for _, r := range m.repos {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (m *Memory) DeleteRepository(_ context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.repos[name]; !ok {
		return ErrNotFound
	}
	delete(m.repos, name)
	for k, a := range m.assets {
		if a.Repository == name {
			delete(m.assets, k)
		}
	}
	return nil
}

func (m *Memory) PutAsset(_ context.Context, a model.Asset) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.assets[key(a.Repository, a.Path)] = a
	return nil
}

func (m *Memory) GetAsset(_ context.Context, repo, path string) (model.Asset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.assets[key(repo, path)]
	if !ok {
		return model.Asset{}, ErrNotFound
	}
	return a, nil
}

func (m *Memory) ListAssets(_ context.Context, repo string) ([]model.Asset, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []model.Asset
	for _, a := range m.assets {
		if a.Repository == repo {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func (m *Memory) DeleteAsset(_ context.Context, repo, path string) (model.Asset, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.assets[key(repo, path)]
	if !ok {
		return model.Asset{}, ErrNotFound
	}
	delete(m.assets, key(repo, path))
	return a, nil
}
