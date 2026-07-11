package v1

import (
	"encoding/json"
	"net/http"

	"dback/models"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) mountTasks(r chi.Router) {
	r.Get("/", h.listTasks)
	r.Post("/", h.createTask)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.getTask)
		r.Put("/", h.updateTask)
		r.Delete("/", h.deleteTask)
		r.Post("/run", h.runTask)
		r.Get("/runs", h.listTaskRuns)
	})
}

func (h *Handler) listTasks(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	tasks, err := h.App.ListTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, Paginated{Items: tasks, Meta: ListMeta{Total: len(tasks)}})
}

func (h *Handler) getTask(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	task, err := h.App.GetTask(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "task not found")
		return
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var task models.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := h.App.SaveTask(task); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_task", err.Error())
		return
	}
	if h.Triggers != nil {
		_ = h.Triggers.Resync(r.Context())
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusCreated, task)
}

func (h *Handler) updateTask(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	var task models.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	task.ID = chi.URLParam(r, "id")
	if err := h.App.SaveTask(task); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_task", err.Error())
		return
	}
	if h.Triggers != nil {
		_ = h.Triggers.Resync(r.Context())
	}
	setRevisionETag(w, h.App.DataRevision())
	writeJSON(w, http.StatusOK, task)
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if !checkMutationPrecondition(w, r, h.App.DataRevision()) {
		return
	}
	if err := h.App.DeleteTask(chi.URLParam(r, "id")); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "task not found")
		return
	}
	if h.Triggers != nil {
		_ = h.Triggers.Resync(r.Context())
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) runTask(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	if h.Triggers == nil {
		writeError(w, http.StatusServiceUnavailable, "triggers_unavailable", "task triggers not running")
		return
	}
	id := chi.URLParam(r, "id")
	if err := h.Triggers.FireNow(r.Context(), id, nil); err != nil {
		writeError(w, http.StatusBadRequest, "task_run_failed", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]string{"task_id": id, "status": "started"})
}

func (h *Handler) listTaskRuns(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	limit, offset := parseLimitOffset(r, 50, 200)
	runs, err := h.App.ListTaskRuns(chi.URLParam(r, "id"), limit+offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", err.Error())
		return
	}
	page, meta := paginateSlice(runs, limit, offset)
	writeJSON(w, http.StatusOK, Paginated{Items: page, Meta: meta})
}
