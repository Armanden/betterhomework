package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	wikipediaBase   = "https://en.wikipedia.org"
	wikimediaREST   = "https://en.wikipedia.org/api/rest_v1"
	userAgent       = "betterhomework/0.1 (homework flashcard generator)"
	maxSummaryChars = 1200
)

var wikipediaClient = &http.Client{Timeout: 20 * time.Second}

// Wikipedia is a search Source backed by the Wikipedia API. It fetches the
// page introduction (summary) for a subject and turns it into a set of
// question/answer flashcards, plus a definition card.
type Wikipedia struct {
	// MaxCards caps the number of cards generated per search (default 8).
	MaxCards int
}

// Name implements Source.
func (w *Wikipedia) Name() string { return "wikipedia" }

// Search implements Source.
func (w *Wikipedia) Search(ctx context.Context, subject string) ([]Card, error) {
	summary, err := w.fetchSummary(ctx, subject)
	if err != nil {
		return nil, err
	}

	max := w.MaxCards
	if max <= 0 {
		max = 8
	}

	sentences := splitSentences(summary)
	// A definition card using the very first sentence.
	defFront := fmt.Sprintf("What is %s?", subject)
	defBack := sentences[0]

	cards := []Card{{Front: defFront, Back: defBack}}
	for _, s := range sentences[1:] {
		if len(cards) >= max {
			break
		}
		cards = append(cards, Card{
			Front: fmt.Sprintf("What is %s about: %s", subject, short(s)),
			Back:  s,
		})
	}
	return cards, nil
}

// fetchSummary retrieves the plain-text introduction for the given subject.
func (w *Wikipedia) fetchSummary(ctx context.Context, subject string) (string, error) {
	title, err := w.resolveTitle(ctx, subject)
	if err != nil {
		return "", err
	}

	u := fmt.Sprintf("%s/page/summary/%s", wikimediaREST, url.PathEscape(title))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := wikipediaClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("wikipedia summary request failed (%s): %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var doc struct {
		Extract string `json:"extract"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("decoding wikipedia summary: %w", err)
	}

	doc.Extract = cleanSpaces(doc.Extract)
	if doc.Extract == "" {
		return "", fmt.Errorf("wikipedia summary for %q was empty", subject)
	}
	if len(doc.Extract) > maxSummaryChars {
		doc.Extract = doc.Extract[:maxSummaryChars]
	}
	return doc.Extract, nil
}

// resolveTitle finds the canonical Wikipedia title for the subject, so that
// partial names ("photosynthesis") still resolve to the right article.
func (w *Wikipedia) resolveTitle(ctx context.Context, subject string) (string, error) {
	params := url.Values{}
	params.Set("action", "query")
	params.Set("list", "search")
	params.Set("srsearch", subject)
	params.Set("format", "json")
	params.Set("srlimit", "1")

	u := wikipediaBase + "/w/api.php?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := wikipediaClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wikipedia search request failed (%s)", resp.Status)
	}

	var doc struct {
		Query struct {
			Search []struct {
				Title string `json:"title"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return "", fmt.Errorf("decoding wikipedia search: %w", err)
	}
	if len(doc.Query.Search) == 0 {
		return "", fmt.Errorf("no wikipedia article found for %q", subject)
	}
	return doc.Query.Search[0].Title, nil
}

// splitSentences splits text into sentences while keeping reasonably sized
// chunks.
func splitSentences(s string) []string {
	replacer := strings.NewReplacer("\n", " ", "\r", " ")
	s = replacer.Replace(s)
	parts := strings.Split(s, ". ")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasSuffix(p, ".") {
			p += "."
		}
		out = append(out, p)
	}
	if len(out) == 0 {
		out = []string{s}
	}
	return out
}

// cleanSpaces collapses runs of whitespace into a single space.
func cleanSpaces(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// short truncates a sentence for use in a question prompt.
func short(s string) string {
	runes := []rune(s)
	if len(runes) > 80 {
		return string(runes[:77]) + "..."
	}
	return s
}
