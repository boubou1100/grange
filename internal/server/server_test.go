package server_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/boubou1100/grange/internal/format"
	"github.com/boubou1100/grange/internal/format/raw"
	"github.com/boubou1100/grange/internal/meta"
	"github.com/boubou1100/grange/internal/server"
	"github.com/boubou1100/grange/internal/storage"
)

func newTestServer(t *testing.T, token string) *httptest.Server {
	t.Helper()
	blobs, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reg := format.NewRegistry()
	reg.Register(raw.Format{})
	s := server.New(server.Config{BaseURL: "http://test", AdminToken: token}, reg,
		format.Deps{Blobs: blobs, Meta: meta.NewMemory()})
	ts := httptest.NewServer(s.Handler())
	t.Cleanup(ts.Close)
	return ts
}

func do(t *testing.T, method, url, body, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestRawLifecycle(t *testing.T) {
	ts := newTestServer(t, "secret")

	// création sans jeton refusée
	if c := do(t, "POST", ts.URL+"/service/rest/v1/repositories", `{"name":"files","format":"raw"}`, "").StatusCode; c != 401 {
		t.Fatalf("want 401, got %d", c)
	}
	// création avec jeton
	if c := do(t, "POST", ts.URL+"/service/rest/v1/repositories", `{"name":"files","format":"raw"}`, "secret").StatusCode; c != 201 {
		t.Fatalf("want 201, got %d", c)
	}
	// format inconnu
	if c := do(t, "POST", ts.URL+"/service/rest/v1/repositories", `{"name":"x","format":"nope"}`, "secret").StatusCode; c != 400 {
		t.Fatalf("want 400, got %d", c)
	}
	// upload refusé sans jeton, accepté avec
	if c := do(t, "PUT", ts.URL+"/repository/files/a/b.txt", "hello", "").StatusCode; c != 401 {
		t.Fatalf("want 401, got %d", c)
	}
	if c := do(t, "PUT", ts.URL+"/repository/files/a/b.txt", "hello", "secret").StatusCode; c != 201 {
		t.Fatalf("want 201, got %d", c)
	}
	// téléchargement public
	resp := do(t, "GET", ts.URL+"/repository/files/a/b.txt", "", "")
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 || string(b) != "hello" {
		t.Fatalf("download: %d %q", resp.StatusCode, b)
	}
	// liste des assets
	resp = do(t, "GET", ts.URL+"/service/rest/v1/assets?repository=files", "", "")
	b, _ = io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "a/b.txt") {
		t.Fatalf("assets: %s", b)
	}
	// suppression puis 404
	if c := do(t, "DELETE", ts.URL+"/repository/files/a/b.txt", "", "secret").StatusCode; c != 204 {
		t.Fatalf("want 204, got %d", c)
	}
	if c := do(t, "GET", ts.URL+"/repository/files/a/b.txt", "", "").StatusCode; c != 404 {
		t.Fatalf("want 404, got %d", c)
	}
}

func TestPathTraversalRejected(t *testing.T) {
	ts := newTestServer(t, "")
	do(t, "POST", ts.URL+"/service/rest/v1/repositories", `{"name":"files","format":"raw"}`, "")
	// Go normalise souvent l'URL ; on teste avec un chemin encodé.
	if c := do(t, "PUT", ts.URL+"/repository/files/%2e%2e/evil", "x", "").StatusCode; c == 201 {
		t.Fatalf("path traversal accepted")
	}
}
