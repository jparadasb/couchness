package templates

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/highercomve/couchness/utils/humanize"
	viewmodels "github.com/highercomve/couchness/web/models"
)

//go:embed *.html
var files embed.FS

// Pages that each define a "content" template.
var pageNames = []string{"shows", "show", "movies", "jobs", "config"}

// Renderer renders full pages (layout + content) and htmx partials.
type Renderer struct {
	pages    map[string]*template.Template // key: "shows","show","movies","jobs"
	partials *template.Template            // layout.html + partials.html
}

// FuncMap is the set of helpers available to every template.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"bytes": func(n int64) string {
			if n <= 0 {
				return "-"
			}
			return humanize.Bytes(uint64(n))
		},
		"since": func(t time.Time) string {
			if t.IsZero() {
				return ""
			}
			return time.Since(t).Round(time.Second).String()
		},
		"base": filepath.Base,
		"sen": func(season, episode int) string {
			return fmt.Sprintf("S%02dE%02d", season, episode)
		},
		"join":  strings.Join,
		"lower": strings.ToLower,
		"in": func(list []string, value string) bool {
			return slices.Contains(list, value)
		},
	}
}

// New parses the embedded templates. Returns an error if any file fails to parse.
func New() (*Renderer, error) {
	base := template.New("base").Funcs(FuncMap())
	base, err := base.ParseFS(files, "layout.html", "partials.html")
	if err != nil {
		return nil, err
	}

	partials := base
	pages := make(map[string]*template.Template, len(pageNames))
	for _, name := range pageNames {
		clone, err := base.Clone()
		if err != nil {
			return nil, err
		}
		clone, err = clone.ParseFS(files, name+".html")
		if err != nil {
			return nil, err
		}
		pages[name] = clone
	}

	return &Renderer{pages: pages, partials: partials}, nil
}

// Render executes the "layout" template of the named page with data.
// Renders into a bytes.Buffer first; on error nothing is written to w and the error is returned.
func (r *Renderer) Render(w io.Writer, page string, data viewmodels.Page) error {
	tmpl, ok := r.pages[page]
	if !ok {
		return fmt.Errorf("unknown page %q", page)
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "layout", data); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

// Partial executes a named template from partials.html with data (same buffer-first rule).
func (r *Renderer) Partial(w io.Writer, name string, data interface{}) error {
	var buf bytes.Buffer
	if err := r.partials.ExecuteTemplate(&buf, name, data); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}
