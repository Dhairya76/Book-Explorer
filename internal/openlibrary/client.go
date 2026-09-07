// Package openlibrary is a small client for the (keyless) Open Library API.
package openlibrary

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Open Library descriptions often contain markdown links, HTML tags, backslash
// line-breaks, and a trailing "----------" source note. These clean that up.
var (
	htmlTagRe   = regexp.MustCompile(`<[^>]+>`)
	mdLinkRe    = regexp.MustCompile(`\[([^\]]+)\]\([^)]*\)`)
	trailingRe  = regexp.MustCompile(`(?s)\n\s*-{4,}.*$`)
	bareSlashRe = regexp.MustCompile(`(?m)^\s*\\+\s*$`)
	blankLineRe = regexp.MustCompile(`\n{3,}`)
)

func cleanDescription(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")     // normalize CRLF first
	s = strings.ReplaceAll(s, "\r", "\n")
	s = trailingRe.ReplaceAllString(s, "")     // drop trailing "---- source ..." note
	s = mdLinkRe.ReplaceAllString(s, "$1")      // [text](url) -> text
	s = htmlTagRe.ReplaceAllString(s, "")       // strip <u> etc.
	s = strings.ReplaceAll(s, "\\\n", "\n")     // backslash line-continuations
	s = bareSlashRe.ReplaceAllString(s, "")     // lone "\" lines
	s = blankLineRe.ReplaceAllString(s, "\n\n") // collapse extra blank lines
	return strings.TrimSpace(s)
}

const (
	searchURL   = "https://openlibrary.org/search.json"
	trendingURL = "https://openlibrary.org/trending/daily.json"
	baseURL     = "https://openlibrary.org"
	coverURLfmt = "https://covers.openlibrary.org/b/id/%d-%s.jpg"

	// PageSize is how many results we request per page.
	PageSize = 24
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// Book is a lightweight view of a work used across the app.
type Book struct {
	Key         string // e.g. "/works/OL893415W"
	ID          string // e.g. "OL893415W"
	Title       string
	Authors     []string
	Year        int
	CoverID     int
	Editions    int
	Rating      float64
	EbookAccess string // "public", "borrowable", "printdisabled", "no_ebook"
	IA          string // first Internet Archive id, if any
}

// CoverURL returns a cover image URL for the given size ("S", "M", "L"),
// or "" when the book has no cover.
func (b Book) CoverURL(size string) string {
	if b.CoverID == 0 {
		return ""
	}
	return fmt.Sprintf(coverURLfmt, b.CoverID, size)
}

// AuthorLine renders the author(s) as a single display string.
func (b Book) AuthorLine() string {
	switch len(b.Authors) {
	case 0:
		return "Unknown author"
	case 1, 2:
		return strings.Join(b.Authors, ", ")
	default:
		return fmt.Sprintf("%s, %s +%d more", b.Authors[0], b.Authors[1], len(b.Authors)-2)
	}
}

func (b Book) HasRating() bool   { return b.Rating > 0 }
func (b Book) RatingStr() string { return fmt.Sprintf("%.1f", b.Rating) }

// Readable reports whether the book is a fully public-domain title that can be
// read for free (as opposed to "borrowable", which needs an account).
func (b Book) Readable() bool { return b.EbookAccess == "public" }

// ReadURL returns where to read the book for free.
func (b Book) ReadURL() string {
	if b.IA != "" {
		return "https://archive.org/details/" + b.IA
	}
	return baseURL + b.Key
}

// Detail is a Book plus the fields we can only get from the work endpoint.
type Detail struct {
	Book
	Description string
	Subjects    []string
	Published   string
	CoverIDs    []int // all cover IDs, so the client can fall back if one fails
}

// doc matches the shape of a search/trending result item.
type doc struct {
	Key            string   `json:"key"`
	Title          string   `json:"title"`
	AuthorName     []string `json:"author_name"`
	FirstPublish   int      `json:"first_publish_year"`
	CoverI         int      `json:"cover_i"`
	EditionCount   int      `json:"edition_count"`
	RatingsAverage float64  `json:"ratings_average"`
	EbookAccess    string   `json:"ebook_access"`
	IA             []string `json:"ia"`
}

func (d doc) toBook() Book {
	b := Book{
		Key:         d.Key,
		ID:          strings.TrimPrefix(d.Key, "/works/"),
		Title:       d.Title,
		Authors:     d.AuthorName,
		Year:        d.FirstPublish,
		CoverID:     d.CoverI,
		Editions:    d.EditionCount,
		Rating:      d.RatingsAverage,
		EbookAccess: d.EbookAccess,
	}
	if len(d.IA) > 0 {
		b.IA = d.IA[0]
	}
	return b
}

// SearchResult is a page of search results.
type SearchResult struct {
	Books    []Book
	Total    int
	Page     int
	PageSize int
}

func getJSON(ctx context.Context, u string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "BookExplorer/1.0 (portfolio project)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("open library returned status %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Search queries Open Library for books matching query. sort may be "" (relevance)
// or one of the Open Library sort keys ("new", "rating", "editions").
func Search(ctx context.Context, query string, page int, sort string) (SearchResult, error) {
	if page < 1 {
		page = 1
	}
	q := url.Values{}
	q.Set("q", query)
	q.Set("page", strconv.Itoa(page))
	q.Set("limit", strconv.Itoa(PageSize))
	q.Set("fields", "key,title,author_name,first_publish_year,cover_i,edition_count,ratings_average,ebook_access,ia")
	if sort != "" {
		q.Set("sort", sort)
	}

	var body struct {
		NumFound int   `json:"numFound"`
		Docs     []doc `json:"docs"`
	}
	if err := getJSON(ctx, searchURL+"?"+q.Encode(), &body); err != nil {
		return SearchResult{}, err
	}

	books := make([]Book, 0, len(body.Docs))
	for _, d := range body.Docs {
		books = append(books, d.toBook())
	}
	return SearchResult{Books: books, Total: body.NumFound, Page: page, PageSize: PageSize}, nil
}

// Trending returns the books trending on Open Library today.
func Trending(ctx context.Context) ([]Book, error) {
	q := url.Values{}
	q.Set("limit", strconv.Itoa(PageSize))

	var body struct {
		Works []doc `json:"works"`
	}
	if err := getJSON(ctx, trendingURL+"?"+q.Encode(), &body); err != nil {
		return nil, err
	}

	books := make([]Book, 0, len(body.Works))
	for _, d := range body.Works {
		books = append(books, d.toBook())
	}
	return books, nil
}

// Work fetches the detail record for a single work (id like "OL893415W").
func Work(ctx context.Context, id string) (Detail, error) {
	workKey := "/works/" + id

	var raw struct {
		Title       string          `json:"title"`
		Description json.RawMessage `json:"description"`
		Subjects    []string        `json:"subjects"`
		Covers      []int           `json:"covers"`
		Published   string          `json:"first_publish_date"`
		Authors     []struct {
			Author struct {
				Key string `json:"key"`
			} `json:"author"`
		} `json:"authors"`
	}
	if err := getJSON(ctx, baseURL+workKey+".json", &raw); err != nil {
		return Detail{}, err
	}

	d := Detail{Published: raw.Published, Description: cleanDescription(parseDescription(raw.Description))}
	d.Key = workKey
	d.ID = id
	d.Title = raw.Title
	if len(raw.Subjects) > 12 {
		d.Subjects = raw.Subjects[:12]
	} else {
		d.Subjects = raw.Subjects
	}
	d.CoverIDs = raw.Covers
	if len(raw.Covers) > 0 {
		d.CoverID = raw.Covers[0]
	}

	// Resolve up to three author names (each is a separate reference).
	for i, a := range raw.Authors {
		if i >= 3 {
			break
		}
		if name := authorName(ctx, a.Author.Key); name != "" {
			d.Authors = append(d.Authors, name)
		}
	}

	return d, nil
}

// Open Library's "description" is sometimes a plain string and sometimes an
// object of the form {"type": "...", "value": "..."}.
func parseDescription(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var obj struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(raw, &obj); err == nil {
		return obj.Value
	}
	return ""
}

func authorName(ctx context.Context, key string) string {
	if key == "" {
		return ""
	}
	var a struct {
		Name string `json:"name"`
	}
	if err := getJSON(ctx, baseURL+key+".json", &a); err != nil {
		return ""
	}
	return a.Name
}
