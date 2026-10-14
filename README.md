# betterhomework

A program that generates and monitors homework, written in Go. This project exists because my school's current homework app is crap and bloated.

The basic idea: a student types the name of a subject into the search bar, the program searches the web for it, and automatically builds an Anki flashcard deck (`.apkg`) from the information it finds.

## How it works

1. The student types a subject in the web UI.
2. The app searches the web for that subject using pluggable search sources (Wikipedia by default).
3. The found text is turned into question/answer flashcards.
4. The cards are exported as a ready-to-import Anki deck (`.apkg`) the student can download and open in Anki.

## Run it

```sh
go run . -addr 127.0.0.1:8080 -out decks
```

Then open http://127.0.0.1:8080 in a browser.

### Flags

| Flag    | Default         | Description                                  |
|---------|-----------------|----------------------------------------------|
| `-addr` | `127.0.0.1:8080`| Address the web UI listens on                |
| `-out`  | `decks`         | Directory where generated `.apkg` files go   |

## Project layout

- `internal/search/` – pluggable search sources (`Source` interface). `wikipedia.go` is the default implementation; add a new file implementing `Source` and register it in `main.go`.
- `internal/anki/` – writes Anki `.apkg` packages using the `genanki-go` library (a zip containing a SQLite `collection.anki2` database).
- `internal/web/` – HTTP server, request handlers, and HTML templates.
- `main.go` – entry point that wires everything together.

> Note: `genanki-go` builds on `go-sqlite3`, which needs a C compiler (CGO) to build.


## Tests

```sh
go test ./...
```
