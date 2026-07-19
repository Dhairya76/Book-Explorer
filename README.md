# 📚 Book Explorer

A server-rendered book search app built with **Go** (standard library only) and
the **[Open Library API](https://openlibrary.org/developers/api)**. Browse
trending titles, search by book / author / subject, and open a detail page with
description and subjects.

**No API key required** — Open Library is fully open, so this runs and deploys
with zero configuration.

## Features

- 🔎 Full-text search (books, authors, subjects) with pagination
- 📈 Trending-today feed on the home page
- 📖 Book detail pages (cover, authors, description, subjects)
- ⚡ Pure Go standard library — no third-party dependencies
- 📦 Single self-contained binary (templates + CSS are embedded via `embed`)
- 🎨 Responsive dark UI

## Tech

Go · `net/http` (1.22 routing) · `html/template` · `embed` · Open Library REST API

## Run locally

```bash
go run .
# open http://localhost:8080
```

Or build a binary:

```bash
go build -o book-explorer .
./book-explorer
```

The server listens on `$PORT` (default `8080`).

## Project layout

```
main.go                     HTTP server, routing, template rendering
internal/openlibrary/       Open Library API client (search, trending, work)
templates/                  layout.html, index.html, book.html (embedded)
static/style.css            styles (embedded)
Dockerfile                  container build for deployment
```

## Deploy free

Because everything is embedded in one binary, deployment is simple:

**Render / Railway / Fly.io** — point them at this repo. A `Dockerfile` is
included; the app reads `$PORT` automatically.

```bash
# Local container test:
docker build -t book-explorer .
docker run -p 8080:8080 book-explorer
```

**Fly.io** quick start:

```bash
fly launch      # detects the Dockerfile
fly deploy
```

## Ideas for next iterations

- Favorites via cookies (add/remove, a `/favorites` page)
- In-memory caching of trending + work lookups
- Author pages and subject browsing
