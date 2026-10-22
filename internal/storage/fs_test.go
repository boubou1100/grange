package storage_test

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/boubou1100/grange/internal/storage"
)

func TestFSDedupAndRead(t *testing.T) {
	s, err := storage.NewFS(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	a, _ := s.Put(ctx, strings.NewReader("same"))
	b, _ := s.Put(ctx, strings.NewReader("same"))
	if a.SHA256 != b.SHA256 || a.Size != 4 {
		t.Fatalf("dedup failed: %+v %+v", a, b)
	}
	rc, err := s.Get(ctx, a.SHA256)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != "same" {
		t.Fatalf("got %q", data)
	}
	if _, err := s.Get(ctx, strings.Repeat("0", 64)); err != storage.ErrNotFound {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
	if _, err := s.Get(ctx, "../etc/passwd"); err == nil {
		t.Fatal("invalid hash accepted")
	}
}
