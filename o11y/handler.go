package o11y

import "log/slog"

// OTELErrorHandler implements the otel.ErrorHandler interface
type OTELErrorHandler struct {
	log *slog.Logger
}

func NewOTELErrorHandler(log *slog.Logger) *OTELErrorHandler {
	return &OTELErrorHandler{log: log}
}

func (h *OTELErrorHandler) Handle(err error) {
	h.log.Error("OTEL export error",
		"error", err,
		"engine", "otel",
		"type", "metric_export_failed",
	)
}
