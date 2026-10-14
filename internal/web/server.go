package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/macro/betterhomework/internal/anki"
	"github.com/macro/betterhomework/internal/search"
)

// Server serves the web interface.
type Server struct {
	registry   *search.Registry
	outputDir  string
	maxResults int
}

// NewServer returns a Server that uses the registry to search and writes
// generated .apkg files into outputDir.
func NewServer(registry *search.Registry, outputDir string) *Server {
	return &Server{registry: registry, outputDir: outputDir, maxResults: 20}
}

// Handler returns the root HTTP handler.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/search", s.handleSearch)
	mux.HandleFunc("/download", s.handleDownload)
	return mux
}

type indexData struct {
	Title   string
	Sources []string
	Error   string
	Cards   []search.Card
	Deck    string
	Subject string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.renderIndex(w, indexData{
		Title:   "betterhomework",
		Sources: s.registry.Sources(),
		Deck:    r.URL.Query().Get("deck"),
		Subject: r.URL.Query().Get("subject"),
	})
}

const deckNamePrefix = "betterhomework_"

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	subject := strings.TrimSpace(r.FormValue("subject"))
	if subject == "" {
		s.renderIndex(w, indexData{Title: "betterhomework", Sources: s.registry.Sources(), Error: "Please enter a subject to search for."})
		return
	}

	ctx := r.Context()
	cards, err := s.registry.Lookup(ctx, subject)
	if err != nil {
		s.renderIndex(w, indexData{Title: "betterhomework", Sources: s.registry.Sources(), Error: err.Error()})
		return
	}

	if len(cards) > s.maxResults {
		cards = cards[:s.maxResults]
	}

	deckName := deckNamePrefix + sanitizeFilename(subject)
	deck := &anki.Deck{Name: deckName}
	for _, c := range cards {
		deck.Cards = append(deck.Cards, anki.Card{Front: c.Front, Back: c.Back})
	}

	path := filepath.Join(s.outputDir, deckName+".apkg")
	if err := deck.WriteAPKG(path); err != nil {
		s.renderIndex(w, indexData{Title: "betterhomework", Sources: s.registry.Sources(), Error: "failed to create deck: " + err.Error()})
		return
	}

	data := indexData{
		Title:   "betterhomework",
		Sources: s.registry.Sources(),
		Cards:   cards,
		Deck:    deckName,
		Subject: subject,
	}
	s.renderIndex(w, data)
}

func (s *Server) renderIndex(w http.ResponseWriter, data indexData) {
	render(w, "index.html", data)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	deck := r.URL.Query().Get("deck")
	if deck == "" {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.outputDir, sanitizeFilename(deck)+".apkg")
	if !strings.HasPrefix(deck, deckNamePrefix) || !fileExists(path) {
		http.NotFound(w, r)
		return
	}

	cd := "attachment; filename=\"" + deck + ".apkg\""
	w.Header().Set("Content-Disposition", cd)
	http.ServeFile(w, r, path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func sanitizeFilename(s string) string {
	replacer := strings.NewReplacer(
		"/", "_", "\\", "_", ":", "_", "*", "_", "?", "_",
		"\"", "_", "<", "_", ">", "_", "|", "_", " ", "_",
	)
	out := replacer.Replace(s)
	out = strings.Trim(out, "_")
	if out == "" {
		out = "deck"
	}
	return out
}
