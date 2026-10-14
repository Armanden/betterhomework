package search

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Source is a pluggable provider that can look up information about a
// subject and turn it into flashcards.
type Source interface {
	// Name returns the display name of the source.
	Name() string
	// Search returns a list of cards for the given subject. Implementations
	// should return a non-nil error if no usable content could be found.
	Search(ctx context.Context, subject string) ([]Card, error)
}

// Card is a single question/answer flashcard.
type Card struct {
	Front string
	Back  string
}

// Registry holds all registered search sources.
type Registry struct {
	sources map[string]Source
	order   []string
}

// NewRegistry returns an empty registry and registers the given sources.
func NewRegistry(sources ...Source) *Registry {
	r := &Registry{sources: make(map[string]Source)}
	for _, s := range sources {
		r.Register(s)
	}
	return r
}

// Register adds a source to the registry.
func (r *Registry) Register(s Source) {
	if s == nil {
		return
	}
	if _, ok := r.sources[s.Name()]; !ok {
		r.order = append(r.order, s.Name())
	}
	r.sources[s.Name()] = s
}

// Sources returns the registered source names in registration order.
func (r *Registry) Sources() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	sort.Strings(out)
	return out
}

// Lookup searches all registered sources and returns a combined, deduplicated
// set of cards. Sources that fail are skipped; an error is returned only when
// no source produces any cards at all.
func (r *Registry) Lookup(ctx context.Context, subject string) ([]Card, error) {
	var (
		cards   []Card
		failures []string
	)
	for _, name := range r.order {
		s := r.sources[name]
		got, err := s.Search(ctx, subject)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", s.Name(), err))
			continue
		}
		cards = append(cards, got...)
	}
	if len(cards) == 0 {
		if len(failures) > 0 {
			return nil, fmt.Errorf("all sources failed: %s", strings.Join(failures, "; "))
		}
		return nil, fmt.Errorf("no cards generated for %q", subject)
	}
	return dedupe(cards), nil
}

// dedupe removes duplicate cards based on the front text.
func dedupe(cards []Card) []Card {
	seen := make(map[string]struct{}, len(cards))
	out := make([]Card, 0, len(cards))
	for _, c := range cards {
		key := strings.ToLower(strings.TrimSpace(c.Front))
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, c)
	}
	return out
}
