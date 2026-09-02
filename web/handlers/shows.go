package handlers

import (
	"net/http"
	"strconv"

	"github.com/highercomve/couchness/models"
	"github.com/highercomve/couchness/web/jobs"
	viewmodels "github.com/highercomve/couchness/web/models"
)

func (h *Handler) shows(w http.ResponseWriter, r *http.Request) {
	list, err := h.Shows.List()
	if err != nil {
		h.fail(w, http.StatusInternalServerError, err)
		return
	}
	data := viewmodels.ShowsPage{Shows: list, Dirs: h.Config.ShowsDirs()}
	if isHX(r) {
		h.partial(w, http.StatusOK, "shows_list", data)
		return
	}
	h.render(w, http.StatusOK, "shows", h.page("Shows", "shows", data))
}

func (h *Handler) scan(w http.ResponseWriter, r *http.Request) {
	h.startJob(w, "scan", h.Library.Scan)
}

func (h *Handler) updateAll(w http.ResponseWriter, r *http.Request) {
	h.startJob(w, "update-all", h.Library.UpdateAll)
}

func (h *Handler) show(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	show, err := h.Shows.Get(id)
	if err != nil {
		h.fail(w, http.StatusNotFound, err)
		return
	}
	data := viewmodels.ShowPage{Show: show}
	h.render(w, http.StatusOK, "show", h.page(show.Title, "shows", data))
}

func (h *Handler) showDownload(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.startJob(w, "download "+id, func(log jobs.Logger) error {
		return h.Library.DownloadShow(id, log)
	})
}

func (h *Handler) showUpdate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	h.startJob(w, "update "+id, func(log jobs.Logger) error {
		return h.Library.UpdateShow(id, log)
	})
}

func (h *Handler) showDisable(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Shows.Disable(id); err != nil {
		h.fail(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	h.flash(w, http.StatusOK, "ok", "Show set to manual")
}

func (h *Handler) showConfig(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		h.fail(w, http.StatusBadRequest, err)
		return
	}
	since := 0
	if value, err := strconv.Atoi(r.FormValue("since")); err == nil {
		since = value
	}
	request := viewmodels.ShowConfigRequest{
		FollowType: r.FormValue("follow_type"),
		Since:      since,
		Resolution: r.FormValue("resolution"),
		Quality:    r.FormValue("quality"),
		Codec:      r.FormValue("codec"),
		Services:   r.Form["services"],
	}
	if _, err := h.Shows.UpdateConfiguration(id, request); err != nil {
		h.fail(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	h.flash(w, http.StatusOK, "ok", "Configuration saved")
}

func (h *Handler) showDelete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	show, err := h.Shows.Delete(id)
	if err != nil {
		h.fail(w, http.StatusNotFound, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	h.flash(w, http.StatusOK, "ok", "Removed "+show.Title+" from tracking")
}

func (h *Handler) identifySearch(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	q := r.URL.Query().Get("q")
	results, err := h.Shows.SearchIMDb(q)
	data := viewmodels.ShowPage{
		Show:    &models.Show{ID: id},
		Query:   q,
		Results: results,
	}
	if err != nil {
		data.Error = err.Error()
	}
	h.partial(w, http.StatusOK, "identify_results", data)
}

func (h *Handler) identify(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := r.ParseForm(); err != nil {
		h.fail(w, http.StatusBadRequest, err)
		return
	}
	externalID := r.FormValue("external_id")
	title := r.FormValue("title")
	if _, err := h.Shows.Identify(id, title, externalID); err != nil {
		h.fail(w, http.StatusBadRequest, err)
		return
	}
	w.Header().Set("HX-Refresh", "true")
	h.flash(w, http.StatusOK, "ok", "Linked to "+externalID)
}
