package commandbus

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/saitadikonda99/deployOS/internal/auth"
	"github.com/saitadikonda99/deployOS/pkg/api"
	"github.com/saitadikonda99/deployOS/pkg/types"
)

// defaultSendTimeout bounds how long an HTTP request waits for a
// device to respond to a command before the caller sees a timeout.
const defaultSendTimeout = 15 * time.Second

// OwnershipChecker reports whether a device belongs to a user.
// internal/devices.Service implements this; Handler depends only on
// this interface so it doesn't need to import internal/devices.
type OwnershipChecker interface {
	IsOwner(ctx context.Context, userID string, deviceID types.AgentID) (bool, error)
}

// Handler adapts Service to HTTP: authenticating callers, checking that
// the target device belongs to them, and translating Service errors
// into status codes.
type Handler struct {
	service       *Service
	authenticator auth.Authenticator
	ownership     OwnershipChecker
	logger        *slog.Logger
}

// NewHandler builds a Handler.
func NewHandler(service *Service, authenticator auth.Authenticator, ownership OwnershipChecker, logger *slog.Logger) *Handler {
	return &Handler{service: service, authenticator: authenticator, ownership: ownership, logger: logger}
}

// Send handles POST /api/v1/devices/{deviceID}/commands.
func (h *Handler) Send(w http.ResponseWriter, r *http.Request) {
	token, ok := auth.BearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	user, err := h.authenticator.Authenticate(r.Context(), token)
	if err != nil {
		if !errors.Is(err, auth.ErrInvalidToken) {
			h.logger.Error("authenticating request", slog.Any("error", err))
		}
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	deviceID := types.AgentID(r.PathValue("deviceID"))
	if err := deviceID.Validate(); err != nil {
		writeError(w, http.StatusBadRequest, "invalid device id")
		return
	}

	owned, err := h.ownership.IsOwner(r.Context(), user.ID, deviceID)
	if err != nil {
		h.logger.Error("checking device ownership", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if !owned {
		writeError(w, http.StatusNotFound, "device not found")
		return
	}

	var req api.SendCommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Kind == "" {
		writeError(w, http.StatusBadRequest, "kind must not be empty")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), defaultSendTimeout)
	defer cancel()

	resp, err := h.service.Send(ctx, deviceID, Request{Kind: req.Kind})
	switch {
	case errors.Is(err, ErrDeviceNotConnected):
		writeError(w, http.StatusServiceUnavailable, "device is not connected")
	case errors.Is(err, context.DeadlineExceeded):
		writeError(w, http.StatusGatewayTimeout, "command timed out waiting for a response")
	case err != nil:
		h.logger.Error("sending command", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "internal error")
	default:
		writeJSON(w, http.StatusOK, api.SendCommandResponse{
			CommandID: resp.CommandID,
			Success:   resp.Success,
			Message:   resp.Message,
			Details:   resp.Details,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, api.ErrorResponse{Error: message})
}
