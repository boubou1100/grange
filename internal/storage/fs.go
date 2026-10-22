package storage

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FS stocke les blobs sur disque : <root>/ab/cd/<sha256>.
type FS struct{ root string }

func NewFS(root string) (*FS, error) {
	if err := os.MkdirAll(filepath.Join(root, "tmp"), 0o755); err != nil {
		return nil, err
	}
	return &FS{root: root}, nil
}

func (s *FS) path(sum string) (string, error) {
	if len(sum) != 64 {
		return "", fmt.Errorf("invalid sha256 %q", sum)
	}
	if _, err := hex.DecodeString(sum); err != nil {
		return "", fmt.Errorf("invalid sha256 %q", sum)
	}
	return filepath.Join(s.root, sum[0:2], sum[2:4], sum), nil
}

func (s *FS) Put(ctx context.Context, r io.Reader) (BlobRef, error) {
	tmp, err := os.CreateTemp(filepath.Join(s.root, "tmp"), "upload-*")
	if err != nil {
		return BlobRef{}, err
	}
	defer os.Remove(tmp.Name()) // sans effet si déjà renommé

	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), r)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return BlobRef{}, err
	}

	sum := hex.EncodeToString(h.Sum(nil))
	dst, _ := s.path(sum)
	if _, err := os.Stat(dst); err == nil {
		return BlobRef{SHA256: sum, Size: n}, nil // déjà présent
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return BlobRef{}, err
	}
	if err := os.Rename(tmp.Name(), dst); err != nil {
		return BlobRef{}, err
	}
	return BlobRef{SHA256: sum, Size: n}, nil
}

func (s *FS) Get(ctx context.Context, sum string) (io.ReadCloser, error) {
	p, err := s.path(sum)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	return f, err
}

func (s *FS) Exists(ctx context.Context, sum string) (bool, error) {
	p, err := s.path(sum)
	if err != nil {
		return false, err
	}
	_, err = os.Stat(p)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return err == nil, err
}

func (s *FS) Delete(ctx context.Context, sum string) error {
	p, err := s.path(sum)
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
