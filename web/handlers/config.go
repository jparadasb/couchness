package handlers

import (
	"net/http"
	"strings"
)

func (h *Handler) configPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "config", h.page("Configuration", "config", nil))
}

func (h *Handler) configSave(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.fail(w, http.StatusBadRequest, err)
		return
	}
	c := *h.Config.Get()
	c.ShowsDir = strings.TrimSpace(r.FormValue("shows_dir"))
	c.ShowsDirs = splitLines(r.FormValue("shows_dirs"))
	c.MoviesDir = strings.TrimSpace(r.FormValue("movies_dir"))
	c.TransmissionHost = strings.TrimSpace(r.FormValue("transmission_host"))
	c.TransmissionPort = strings.TrimSpace(r.FormValue("transmission_port"))
	c.TransmissionAuth = strings.TrimSpace(r.FormValue("transmission_auth"))
	c.OmdbAPIKey = strings.TrimSpace(r.FormValue("omdb_api_key"))
	if err := h.Config.Save(&c); err != nil {
		h.fail(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	h.flash(w, http.StatusOK, "ok", "Configuration saved")
}

func splitLines(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}
