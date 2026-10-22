// Package model définit le modèle de données commun à tous les formats.
//
// Repository -> contient des Components -> composés d'Assets -> pointant vers des blobs.
// Tous les formats (raw, maven, npm, oci...) se ramènent à ce modèle, ce qui permet
// à l'API, à la recherche, aux politiques de nettoyage et à l'interface web
// de fonctionner sans code spécifique à un format.
package model

import "time"

// RepoType est le type de dépôt (même vocabulaire que Nexus).
type RepoType string

const (
	Hosted RepoType = "hosted" // on publie directement dedans
	Proxy  RepoType = "proxy"  // cache d'un dépôt distant (ex: Maven Central)
	Group  RepoType = "group"  // agrège plusieurs dépôts derrière une URL
)

// Repository est un dépôt configuré par l'administrateur.
type Repository struct {
	Name      string            `json:"name"`
	Format    string            `json:"format"` // "raw", "maven2", "npm"...
	Type      RepoType          `json:"type"`
	URL       string            `json:"url"` // renseignée par le serveur
	Attrs     map[string]string `json:"attributes,omitempty"`
	CreatedAt time.Time         `json:"createdAt"`
}

// Component est l'unité logique : un paquet+version, une image+tag...
type Component struct {
	ID         string `json:"id"`
	Repository string `json:"repository"`
	Format     string `json:"format"`
	Group      string `json:"group,omitempty"`
	Name       string `json:"name"`
	Version    string `json:"version,omitempty"`
}

// Asset est un fichier réel rattaché à un dépôt (et souvent à un component).
// Le contenu est stocké ailleurs, adressé par son hash SHA-256 (déduplication).
type Asset struct {
	Repository  string    `json:"repository"`
	Path        string    `json:"path"`
	ComponentID string    `json:"componentId,omitempty"`
	BlobSHA256  string    `json:"sha256"`
	Size        int64     `json:"size"`
	ContentType string    `json:"contentType,omitempty"`
	UploadedAt  time.Time `json:"uploadedAt"`
}
