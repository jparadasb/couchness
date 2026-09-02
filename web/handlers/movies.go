package handlers

import (
	"net/http"
	"strconv"

	viewmodels "github.com/highercomve/couchness/web/models"
)

func (h *Handler) movies(w http.ResponseWriter, r *http.Request) {
	list, err := h.Movies.List()
	data := viewmodels.MoviesPage{
		Movies:    list,
		MoviesDir: h.Config.MoviesDir(),
	}
	if err != nil {
		data.Error = err.Error()
	}
	h.render(w, http.StatusOK, "movies", h.page("Movies", "movies", data))
}

func (h *Handler) moviesList(w http.ResponseWriter, r *http.Request) {
	list, err := h.Movies.List()
	data := viewmodels.MoviesPage{
		Movies:    list,
		MoviesDir: h.Config.MoviesDir(),
	}
	if err != nil {
		data.Error = err.Error()
	}
	h.partial(w, http.StatusOK, "movies_list", data)
}

func (h *Handler) movieSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	results, err := h.Movies.Search(q)
	data := viewmodels.MovieSearch{Query: q, Results: results}
	if err != nil {
		data.Error = err.Error()
	}
	h.partial(w, http.StatusOK, "movie_results", data)
}

func (h *Handler) movieTorrents(w http.ResponseWriter, r *http.Request) {
	req := movieRequest(r)
	movie, torrents, err := h.Movies.Torrents(req)
	data := viewmodels.MovieTorrents{Movie: movie, Torrents: torrents}
	if err != nil {
		data.Error = err.Error()
	}
	h.partial(w, http.StatusOK, "movie_torrents", data)
}

func (h *Handler) movieDownload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.fail(w, http.StatusBadRequest, err)
		return
	}
	movie, err := h.Movies.Download(movieRequest(r), torrentRequest(r))
	if err != nil {
		status := http.StatusInternalServerError
		if movie == nil {
			status = http.StatusBadRequest
		}
		h.fail(w, status, err)
		return
	}
	w.Header().Set("HX-Trigger", "movies-changed")
	h.flash(w, http.StatusOK, "ok", "Queued "+movie.Show.Title+" in transmission")
}

func (h *Handler) movieDelete(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.fail(w, http.StatusBadRequest, err)
		return
	}
	movie, err := h.Movies.Delete(r.FormValue("key"))
	if err != nil {
		h.fail(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("HX-Trigger", "movies-changed")
	h.flash(w, http.StatusOK, "ok", "Removed "+movie.Show.Title)
}

func movieRequest(r *http.Request) viewmodels.MovieRequest {
	return viewmodels.MovieRequest{
		ImdbID: r.FormValue("imdb_id"),
		Title:  r.FormValue("title"),
		Year:   r.FormValue("year"),
		Key:    r.FormValue("key"),
		Folder: r.FormValue("folder"),
	}
}

func torrentRequest(r *http.Request) viewmodels.TorrentRequest {
	size, _ := strconv.ParseInt(r.FormValue("size"), 10, 64)
	seeds, _ := strconv.Atoi(r.FormValue("seeds"))
	return viewmodels.TorrentRequest{
		Title:      r.FormValue("torrent_title"),
		Magnet:     r.FormValue("magnet"),
		Size:       size,
		Seeds:      seeds,
		Resolution: r.FormValue("resolution"),
		Quality:    r.FormValue("quality"),
		Codec:      r.FormValue("codec"),
	}
}
