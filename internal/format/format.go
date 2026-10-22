// Package format définit l'interface de plugin de format.
//
// Ajouter un format (maven2, npm, pypi, oci...) = implémenter Format et
// l'enregistrer dans le Registry. Rien à toucher dans l'API ni dans l'UI :
// l'interface web se construit à partir de UIDescriptor.
package format

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/boubou1100/grange/internal/meta"
	"github.com/boubou1100/grange/internal/model"
	"github.com/boubou1100/grange/internal/storage"
)

// Deps regroupe ce dont un format a besoin.
type Deps struct {
	Blobs storage.BlobStore
	Meta  meta.Store
}

// ClientSnippet est une commande prête à copier ("Se connecter à ce dépôt").
type ClientSnippet struct {
	Title string `json:"title"`
	Body  string `json:"body"` // peut contenir {{url}} et {{name}}
}

// UIDescriptor décrit comment l'interface doit présenter le format.
type UIDescriptor struct {
	DisplayName string          `json:"displayName"`
	Snippets    []ClientSnippet `json:"snippets"`
}

// Format est un plugin de format de paquet.
type Format interface {
	// Name est l'identifiant technique ("raw", "maven2", "npm"...).
	Name() string
	// Handler expose le protocole natif pour un dépôt donné. Les chemins reçus
	// sont relatifs à la racine du dépôt (/repository/<name>/ est déjà retiré).
	Handler(repo model.Repository, deps Deps) http.Handler
	// UI renvoie le descripteur utilisé par l'interface web.
	UI() UIDescriptor
}

// Registry contient les formats disponibles.
type Registry struct{ formats map[string]Format }

func NewRegistry() *Registry { return &Registry{formats: map[string]Format{}} }

func (r *Registry) Register(f Format) {
	if _, dup := r.formats[f.Name()]; dup {
		panic(fmt.Sprintf("format %q registered twice", f.Name()))
	}
	r.formats[f.Name()] = f
}

func (r *Registry) Get(name string) (Format, bool) {
	f, ok := r.formats[name]
	return f, ok
}

func (r *Registry) Names() []string {
	out := make([]string, 0, len(r.formats))
	for n := range r.formats {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
