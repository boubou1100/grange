// Package server assemble les routes HTTP : API REST (compatible Nexus) et
// protocoles de formats.
package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/boubou1100/grange/internal/format"
	"github.com/boubou1100/grange/internal/meta"
	"github.com/boubou1100/grange/internal/model"
)

type Config struct {
	BaseURL    string // ex: http://localhost:8081
	AdminToken string // si vide, aucune authentification (dev uniquement !)
}

type Server struct {
	cfg  Config
	reg  *format.Registry
	deps format.Deps
	mux  *http.ServeMux
}

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,62}$`)

func New(cfg Config, reg *format.Registry, deps format.Deps) *Server {
	s := &Server{cfg: cfg, reg: reg, deps: deps, mux: http.NewServeMux()}

	// API compatible Nexus (sous-ensemble) : les scripts existants fonctionnent.
	s.mux.HandleFunc("GET /service/rest/v1/status", s.status)
	s.mux.HandleFunc("GET /service/rest/v1/repositories", s.listRepos)
	s.mux.HandleFunc("POST /service/rest/v1/repositories", s.auth(s.createRepo))
	s.mux.HandleFunc("DELETE /service/rest/v1/repositories/{name}", s.auth(s.deleteRepo))
	s.mux.HandleFunc("GET /service/rest/v1/assets", s.listAssets)

	// API propre à Grange pour l'UI (formats disponibles + snippets).
	s.mux.HandleFunc("GET /api/v1/formats", s.listFormats)

	// Protocoles natifs des formats.
	s.mux.HandleFunc("/repository/{name}/{path...}", s.repository)
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

// auth protège les écritures par un jeton Bearer. Placeholder : le vrai système
// (utilisateurs, rôles par dépôt, OIDC/LDAP) viendra ensuite.
func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.cfg.AdminToken != "" {
			if r.Header.Get("Authorization") != "Bearer "+s.cfg.AdminToken {
				w.Header().Set("WWW-Authenticate", `Bearer realm="grange"`)
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) repoURL(name string) string {
	return strings.TrimRight(s.cfg.BaseURL, "/") + "/repository/" + name + "/"
}

func (s *Server) status(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }

func (s *Server) listRepos(w http.ResponseWriter, r *http.Request) {
	repos, err := s.deps.Meta.ListRepositories(r.Context())
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	for i := range repos {
		repos[i].URL = s.repoURL(repos[i].Name)
	}
	writeJSON(w, http.StatusOK, repos)
}

type createReq struct {
	Name   string         `json:"name"`
	Format string         `json:"format"`
	Type   model.RepoType `json:"type"`
}

func (s *Server) createRepo(w http.ResponseWriter, r *http.Request) {
	var req createReq
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		req.Type = model.Hosted
	}
	switch {
	case !nameRe.MatchString(req.Name):
		http.Error(w, "invalid repository name", http.StatusBadRequest)
		return
	case req.Type != model.Hosted: // proxy et group : à venir
		http.Error(w, "only hosted repositories are supported yet", http.StatusNotImplemented)
		return
	}
	if _, ok := s.reg.Get(req.Format); !ok {
		http.Error(w, "unknown format; available: "+strings.Join(s.reg.Names(), ", "), http.StatusBadRequest)
		return
	}
	repo := model.Repository{Name: req.Name, Format: req.Format, Type: req.Type, CreatedAt: time.Now().UTC()}
	if err := s.deps.Meta.CreateRepository(r.Context(), repo); errors.Is(err, meta.ErrExists) {
		http.Error(w, "repository already exists", http.StatusConflict)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	repo.URL = s.repoURL(repo.Name)
	writeJSON(w, http.StatusCreated, repo)
}

func (s *Server) deleteRepo(w http.ResponseWriter, r *http.Request) {
	if err := s.deps.Meta.DeleteRepository(r.Context(), r.PathValue("name")); errors.Is(err, meta.ErrNotFound) {
		http.NotFound(w, r)
		return
	} else if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listAssets(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repository")
	if repo == "" {
		http.Error(w, "repository parameter required", http.StatusBadRequest)
		return
	}
	assets, err := s.deps.Meta.ListAssets(r.Context(), repo)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": assets, "continuationToken": nil})
}

func (s *Server) listFormats(w http.ResponseWriter, r *http.Request) {
	out := map[string]format.UIDescriptor{}
	for _, n := range s.reg.Names() {
		f, _ := s.reg.Get(n)
		out[n] = f.UI()
	}
	writeJSON(w, http.StatusOK, out)
}

// repository route une requête vers le plugin du format du dépôt.
func (s *Server) repository(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	repo, err := s.deps.Meta.GetRepository(r.Context(), name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	f, ok := s.reg.Get(repo.Format)
	if !ok {
		http.Error(w, "format not available", http.StatusInternalServerError)
		return
	}
	// Les écritures exigent le jeton (si configuré) ; les lectures sont publiques en dev.
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		var denied bool
		s.auth(func(http.ResponseWriter, *http.Request) {})(&denyRecorder{w, &denied}, r)
		if denied {
			return
		}
	}
	r2 := r.Clone(r.Context())
	r2.URL.Path = "/" + r.PathValue("path")
	f.Handler(repo, s.deps).ServeHTTP(w, r2)
}

// denyRecorder détecte si auth() a répondu (donc refusé).
type denyRecorder struct {
	http.ResponseWriter
	denied *bool
}

func (d *denyRecorder) WriteHeader(code int) { *d.denied = true; d.ResponseWriter.WriteHeader(code) }
func (d *denyRecorder) Write(b []byte) (int, error) {
	*d.denied = true
	return d.ResponseWriter.Write(b)
}
