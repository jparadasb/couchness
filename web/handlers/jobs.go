package handlers

import (
	"net/http"
)

func (h *Handler) jobsPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "jobs", h.page("Jobs", "jobs", h.Jobs.List()))
}

func (h *Handler) job(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	j, ok := h.Jobs.Get(id)
	if !ok {
		h.flash(w, http.StatusNotFound, "error", "job not found")
		return
	}
	snap := j.Snapshot()
	if snap.Done {
		w.Header().Set("HX-Trigger", "job-finished")
	}
	h.partial(w, http.StatusOK, "job", snap)
}
