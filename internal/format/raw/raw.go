// Package raw implémente le format "raw" : fichiers bruts adressés par chemin.
//
//	PUT    /repository/<repo>/<chemin>   upload
//	GET    /repository/<repo>/<chemin>   téléchargement
//	HEAD   /repository/<repo>/<chemin>
//	DELETE /repository/<repo>/<chemin>
//
// C'est le format le plus simple : il valide toute la chaîne (stockage, métadonnées,
// auth, API) avant d'attaquer Maven, npm, etc.
package raw

import (
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/boubou1100/grange/internal/format"
	"github.com/boubou1100/grange/internal/meta"
	"github.com/boubou1100/grange/internal/model"
	"github.com/boubou1100/grange/internal/storage"
)

type Format struct{}

func (Format) Name() string { return "raw" }

func (Format) UI() format.UIDescriptor {
	return format.UIDescriptor{
		DisplayName: "Raw (fichiers bruts)",
		Snippets: []format.ClientSnippet{
			{Title: "Envoyer un fichier", Body: "curl -u USER:TOKEN --upload-file ./fichier.zip {{url}}fichier.zip"},
			{Title: "Télécharger un fichier", Body: "curl -O {{url}}fichier.zip"},
		},
	}
}

func (Format) Handler(repo model.Repository, d format.Deps) http.Handler {
	return &handler{repo: repo, d: d}
}

type handler struct {
	repo model.Repository
	d    format.Deps
}

// clean normalise le chemin et refuse toute tentative de remontée (../).
func clean(p string) (string, bool) {
	p = strings.TrimPrefix(p, "/")
	if p == "" || strings.Contains(p, "\x00") {
		return "", false
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", false
		}
	}
	return path.Clean(p), true
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	p, ok := clean(r.URL.Path)
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		h.get(w, r, p)
	case http.MethodPut, http.MethodPost:
		if h.repo.Type != model.Hosted {
			http.Error(w, "repository is not hosted", http.StatusMethodNotAllowed)
			return
		}
		h.put(w, r, p)
	case http.MethodDelete:
		h.delete(w, r, p)
	default:
		w.Header().Set("Allow", "GET, HEAD, PUT, POST, DELETE")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *handler) put(w http.ResponseWriter, r *http.Request, p string) {
	ref, err := h.d.Blobs.Put(r.Context(), r.Body)
	if err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	err = h.d.Meta.PutAsset(r.Context(), model.Asset{
		Repository:  h.repo.Name,
		Path:        p,
		BlobSHA256:  ref.SHA256,
		Size:        ref.Size,
		ContentType: r.Header.Get("Content-Type"),
		UploadedAt:  time.Now().UTC(),
	})
	if err != nil {
		http.Error(w, "metadata error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (h *handler) get(w http.ResponseWriter, r *http.Request, p string) {
	a, err := h.d.Meta.GetAsset(r.Context(), h.repo.Name, p)
	if errors.Is(err, meta.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "metadata error", http.StatusInternalServerError)
		return
	}
	ct := a.ContentType
	if ct == "" {
		ct = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Content-Length", strconv.FormatInt(a.Size, 10))
	w.Header().Set("ETag", `"`+a.BlobSHA256+`"`)
	if r.Method == http.MethodHead {
		return
	}
	rc, err := h.d.Blobs.Get(r.Context(), a.BlobSHA256)
	if errors.Is(err, storage.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}
	defer rc.Close()
	_, _ = copyTo(w, rc)
}
