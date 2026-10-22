// Commande grange : serveur de dépôts d'artefacts libre.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/boubou1100/grange/internal/format"
	"github.com/boubou1100/grange/internal/format/raw"
	"github.com/boubou1100/grange/internal/meta"
	"github.com/boubou1100/grange/internal/server"
	"github.com/boubou1100/grange/internal/storage"
)

var version = "dev"

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	if len(os.Args) < 2 || os.Args[1] != "serve" {
		fmt.Fprintf(os.Stderr, "grange %s\n\nusage: grange serve [-addr :8081] [-data ./data]\n", version)
		os.Exit(2)
	}
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	addr := fs.String("addr", env("GRANGE_ADDR", ":8081"), "adresse d'écoute")
	data := fs.String("data", env("GRANGE_DATA", "./data"), "répertoire de stockage des blobs")
	base := fs.String("base-url", env("GRANGE_BASE_URL", "http://localhost:8081"), "URL publique")
	_ = fs.Parse(os.Args[2:])

	blobs, err := storage.NewFS(*data)
	if err != nil {
		log.Fatal(err)
	}

	reg := format.NewRegistry()
	reg.Register(raw.Format{}) // ajouter ici maven2, npm, pypi, oci...

	token := os.Getenv("GRANGE_ADMIN_TOKEN")
	if token == "" {
		log.Println("ATTENTION: GRANGE_ADMIN_TOKEN non défini, les écritures ne sont pas protégées")
	}

	srv := server.New(
		server.Config{BaseURL: *base, AdminToken: token},
		reg,
		format.Deps{Blobs: blobs, Meta: meta.NewMemory()}, // TODO: PostgreSQL
	)

	httpSrv := &http.Server{Addr: *addr, Handler: srv.Handler(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		log.Printf("grange %s écoute sur %s", version, *addr)
		if err := httpSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
}
