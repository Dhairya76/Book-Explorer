package main

import (
	"bytes"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"book-explorer/internal/openlibrary"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

var (
	homeTmpl = template.Must(template.ParseFS(templatesFS, "templates/layout.html", "templates/index.html"))
	bookTmpl = template.Must(template.ParseFS(templatesFS, "templates/layout.html", "templates/book.html"))
)

type homeData struct {
	Query    string
	Sort     string
	IsSearch bool
	Books    []openlibrary.Book
	Total    int
	Page     int
	HasPrev  bool
	HasNext  bool
	PrevPage int
	NextPage int
	Error    string
}

type bookData struct {
	openlibrary.Detail
	Query string
	Error string
}

func main() {
	mux := http.NewServeMux()
	mux.Handle("GET /static/", http.FileServer(http.FS(staticFS)))
	mux.HandleFunc("GET /{$}", handleHome)
	mux.HandleFunc("GET /book/{id}", handleBook)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Book Explorer listening on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// validSort whitelists the sort options we expose, so only known values reach
// the Open Library API.
func validSort(s string) string {
	switch s {
	case "new", "rating", "editions":
		return s
	default:
		return ""
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	sort := validSort(r.URL.Query().Get("sort"))
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	data := homeData{Query: query, Sort: sort, IsSearch: query != "", Page: page}

	if query != "" {
		res, err := openlibrary.Search(r.Context(), query, page, sort)
		if err != nil {
			data.Error = "Couldn't reach Open Library. Please try again."
		} else {
			data.Books = res.Books
			data.Total = res.Total
			totalPages := (res.Total + res.PageSize - 1) / res.PageSize
			data.HasPrev = page > 1
			data.HasNext = page < totalPages
			data.PrevPage = page - 1
			data.NextPage = page + 1
		}
	} else {
		books, err := openlibrary.Trending(r.Context())
		if err != nil {
			data.Error = "Couldn't load trending books. Please try again."
		} else {
			data.Books = books
		}
	}

	render(w, homeTmpl, http.StatusOK, data)
}

func handleBook(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	detail, err := openlibrary.Work(r.Context(), id)
	if err != nil {
		render(w, bookTmpl, http.StatusNotFound, bookData{Error: "Sorry, we couldn't find that book."})
		return
	}
	render(w, bookTmpl, http.StatusOK, bookData{Detail: detail})
}

// render executes the template into a buffer first so a mid-render error can't
// leave a half-written response.
func render(w http.ResponseWriter, t *template.Template, status int, data any) {
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout.html", data); err != nil {
		log.Println("template error:", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}
