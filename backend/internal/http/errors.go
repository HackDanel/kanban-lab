// Package httpserver wires the HTTP API: routing, middleware, request
// validation against the OpenAPI contract and the handlers.
package httpserver

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

var kindStatus = map[domain.Kind]int{
	domain.KindValidation:       http.StatusBadRequest,
	domain.KindUnauthorized:     http.StatusUnauthorized,
	domain.KindForbidden:        http.StatusForbidden,
	domain.KindNotFound:         http.StatusNotFound,
	domain.KindConflict:         http.StatusConflict,
	domain.KindWipLimit:         http.StatusConflict,
	domain.KindColumnNotEmpty:   http.StatusConflict,
	domain.KindRateLimited:      http.StatusTooManyRequests,
	domain.KindFileTooLarge:     http.StatusRequestEntityTooLarge,
	domain.KindUnsupportedMedia: http.StatusUnsupportedMediaType,
	domain.KindPayloadTooLarge:  http.StatusRequestEntityTooLarge,
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

// writeError renders err in the API error format. Unexpected errors are
// logged and reported without details.
func writeError(w http.ResponseWriter, r *http.Request, log *slog.Logger, err error) {
	var de *domain.Error
	if !errors.As(err, &de) {
		log.ErrorContext(r.Context(), "request failed", "method", r.Method, "path", r.URL.Path, "err", err)
		writeJSON(w, http.StatusInternalServerError, api.Error{Code: domain.KindInternal.Code(), Message: "Внутренняя ошибка сервера"})
		return
	}
	body := api.Error{Code: de.Kind.Code(), Message: de.Message}
	if len(de.Fields) > 0 {
		details := make([]api.FieldError, len(de.Fields))
		for i, f := range de.Fields {
			details[i] = api.FieldError{Field: f.Field, Message: f.Message}
		}
		body.Details = &details
	}
	status, ok := kindStatus[de.Kind]
	if !ok {
		status = http.StatusInternalServerError
	}
	writeJSON(w, status, body)
}
