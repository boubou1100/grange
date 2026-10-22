// Package storage définit l'abstraction de stockage de blobs adressés par contenu.
package storage

import (
	"context"
	"errors"
	"io"
)

var ErrNotFound = errors.New("blob not found")

// BlobRef identifie un blob stocké.
type BlobRef struct {
	SHA256 string
	Size   int64
}

// BlobStore stocke des blobs par hash. Implémentations prévues : système de
// fichiers (fourni), S3/MinIO (à venir).
type BlobStore interface {
	// Put lit r jusqu'à la fin, calcule le SHA-256 et stocke le contenu.
	// Si le blob existe déjà, rien n'est réécrit (déduplication).
	Put(ctx context.Context, r io.Reader) (BlobRef, error)
	Get(ctx context.Context, sha256 string) (io.ReadCloser, error)
	Exists(ctx context.Context, sha256 string) (bool, error)
	Delete(ctx context.Context, sha256 string) error
}
