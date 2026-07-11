package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"dback/internal/event"
	"dback/internal/operation"

	"github.com/go-chi/chi/v5"
)

func (h *Handler) mountOperations(r chi.Router) {
	r.Get("/", h.listOperations)
	r.Post("/", h.createOperation)
	r.Get("/stream", h.streamOperations)
	r.Route("/{id}", func(r chi.Router) {
		r.Get("/", h.getOperation)
		r.Post("/cancel", h.cancelOperation)
		r.Post("/retry", h.retryOperation)
		r.Get("/logs", h.operationLogs)
	})
}

func (h *Handler) listOperations(w http.ResponseWriter, r *http.Request) {
	limit, offset := parseLimitOffset(r, 50, 200)
	all := h.CP.Records.List(limit + offset)
	if offset > len(all) {
		all = nil
	} else {
		all = all[offset:]
	}
	if len(all) > limit {
		all = all[:limit]
	}
	items := make([]OperationDTO, 0, len(all))
	for _, rec := range all {
		items = append(items, operationFromRecord(rec))
	}
	writeJSON(w, http.StatusOK, paginatedResponse(items, ListMeta{Total: len(items), Limit: limit, Offset: offset}))
}

type createOperationRequest struct {
	Kind       string          `json:"kind"`
	ProfileID  string          `json:"profile_id"`
	TriggerRef string          `json:"trigger_ref,omitempty"`
	Params     json.RawMessage `json:"params,omitempty"`
}

func (h *Handler) createOperation(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	var req createOperationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	spec, err := operation.SpecFromAction(req.ProfileID, req.TriggerRef, operation.Kind(req.Kind), req.Params)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_operation", err.Error())
		return
	}
	switch p := spec.Params.(type) {
	case operation.RestoreParams:
		spec.ProfileID = p.DestinationProfileID
	case operation.DeepVerifyParams:
		spec.ProfileID = p.DestinationProfileID
	}
	if spec.TriggerRef == "" {
		spec.TriggerRef = "api"
	}
	rec, err := h.CP.Dispatcher.SubmitAsync(h.CP.OperationContext(), spec)
	if err != nil {
		writeError(w, http.StatusConflict, "operation_rejected", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, operationFromRecord(rec))
}

func (h *Handler) getOperation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	rec, err := h.CP.Dispatcher.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "operation not found")
		return
	}
	writeJSON(w, http.StatusOK, operationFromRecord(rec))
}

func (h *Handler) cancelOperation(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.CP.Dispatcher.Cancel(id); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "operation not found")
		return
	}
	rec, _ := h.CP.Dispatcher.Get(id)
	writeJSON(w, http.StatusOK, operationFromRecord(rec))
}

func (h *Handler) retryOperation(w http.ResponseWriter, r *http.Request) {
	if !requireVaultUnlocked(w, h.App) {
		return
	}
	id := chi.URLParam(r, "id")
	prev, err := h.CP.Dispatcher.Get(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "operation not found")
		return
	}
	params, err := operation.DecodeParams(prev.Kind, prev.Params)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_operation", err.Error())
		return
	}
	spec := operation.Spec{
		Kind:       prev.Kind,
		ProfileID:  prev.ProfileID,
		TriggerRef: "retry:" + id,
		Params:     params,
	}
	rec, err := h.CP.Dispatcher.SubmitAsync(h.CP.OperationContext(), spec)
	if err != nil {
		writeError(w, http.StatusConflict, "operation_rejected", err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, operationFromRecord(rec))
}

func (h *Handler) operationLogs(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	limit, offset := parseLimitOffset(r, 100, 500)
	var filtered []map[string]any
	for _, entry := range h.App.Logs() {
		if entry.OperationID != id {
			continue
		}
		filtered = append(filtered, map[string]any{
			"id":        entry.ID,
			"timestamp": entry.Timestamp,
			"action":    entry.Action,
			"phase":     entry.Phase,
			"level":     entry.Level,
			"details":   entry.Details,
			"status":    entry.Status,
			"error":     entry.Error,
		})
	}
	page, meta := paginateSlice(filtered, limit, offset)
	writeJSON(w, http.StatusOK, paginatedResponse(page, meta))
}

func (h *Handler) streamOperations(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "sse_unsupported", "streaming not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ctx := r.Context()
	unsubs := make([]func(), 0, 6)
	send := func(name string, payload any) {
		b, _ := json.Marshal(payload)
		_, _ = w.Write([]byte("event: " + name + "\n"))
		_, _ = w.Write([]byte("data: "))
		_, _ = w.Write(b)
		_, _ = w.Write([]byte("\n\n"))
		flusher.Flush()
	}
	subscribe := func(t event.Type, fn func(context.Context, event.Event)) {
		unsubs = append(unsubs, h.CP.Bus.Subscribe(t, func(c context.Context, ev event.Event) error {
			fn(c, ev)
			return nil
		}))
	}
	subscribeSSE(h, send, subscribe)
	defer func() {
		for _, u := range unsubs {
			u()
		}
	}()

	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			_, _ = w.Write([]byte(": heartbeat\n\n"))
			flusher.Flush()
		}
	}
}

func subscribeSSE(_ *Handler, send func(string, any), subscribe func(event.Type, func(context.Context, event.Event))) {
	subscribe(event.TypeOperationStarted, func(_ context.Context, ev event.Event) {
		e := ev.(event.OperationStarted)
		send("operation.started", map[string]any{
			"operation_id": e.OperationID,
			"kind":         e.Kind,
			"profile_id":   e.ProfileID,
			"timestamp":    e.Timestamp,
		})
	})
	subscribe(event.TypeOperationProgress, func(_ context.Context, ev event.Event) {
		e := ev.(event.OperationProgress)
		send("operation.progress", map[string]any{
			"operation_id": e.OperationID,
			"message":      e.Message,
			"current":      e.Current,
			"total":        e.Total,
			"timestamp":    e.Timestamp,
		})
	})
	subscribe(event.TypeOperationCompleted, func(_ context.Context, ev event.Event) {
		e := ev.(event.OperationCompleted)
		send("operation.completed", sseOperationPayload(e.Envelope, e.Result))
	})
	subscribe(event.TypeOperationFailed, func(_ context.Context, ev event.Event) {
		e := ev.(event.OperationFailed)
		send("operation.failed", sseOperationPayload(e.Envelope, e.Result))
	})
	subscribe(event.TypeOperationCanceled, func(_ context.Context, ev event.Event) {
		e := ev.(event.OperationCanceled)
		send("operation.canceled", sseOperationPayload(e.Envelope, e.Result))
	})
	subscribe(event.TypeTaskSkipped, func(_ context.Context, ev event.Event) {
		e := ev.(event.TaskSkipped)
		send("task.skipped", map[string]any{
			"task_id":    e.TaskID,
			"profile_id": e.ProfileID,
			"reason":     e.Reason,
			"timestamp":  e.Timestamp,
		})
	})
}

func sseOperationPayload(env event.Envelope, res operation.Result) map[string]any {
	return map[string]any{
		"operation_id": env.OperationID,
		"kind":         env.Kind,
		"profile_id":   env.ProfileID,
		"status":       res.Status,
		"error":        res.Error,
		"timestamp":    env.Timestamp,
	}
}
