package search

import (
	"context"
	"errors"
	"testing"
)

type fakeSource struct {
	name  string
	cards []Card
	err   error
}

func (f *fakeSource) Name() string { return f.name }
func (f *fakeSource) Search(_ context.Context, _ string) ([]Card, error) {
	return f.cards, f.err
}

func TestLookupCombinesAndDedupes(t *testing.T) {
	r := NewRegistry(
		&fakeSource{name: "a", cards: []Card{{Front: "X", Back: "1"}, {Front: "Y", Back: "2"}}},
		&fakeSource{name: "b", cards: []Card{{Front: "x", Back: "dup"}}}, // case-insensitive dup of X
		&fakeSource{name: "c", err: errors.New("boom")},
	)
	cards, err := r.Lookup(context.Background(), "subject")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("expected 2 deduped cards, got %d: %+v", len(cards), cards)
	}
	if cards[0].Back != "1" {
		t.Fatalf("expected first card from source a, got %+v", cards[0])
	}
}

func TestLookupAllFailed(t *testing.T) {
	r := NewRegistry(&fakeSource{name: "a", err: errors.New("boom")})
	_, err := r.Lookup(context.Background(), "subj")
	if err == nil {
		t.Fatal("expected error when all sources fail")
	}
}
