package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/macro/betterhomework/internal/search"
	"github.com/macro/betterhomework/internal/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "address to serve the web UI on")
	outDir := flag.String("out", "decks", "directory where generated .apkg files are written")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatalf("creating output dir: %v", err)
	}

	registry := search.NewRegistry(&search.Wikipedia{})

	srv := web.NewServer(registry, *outDir)

	log.Printf("betterhomework running at http://%s", *addr)
	if err := http.ListenAndServe(*addr, srv.Handler()); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
