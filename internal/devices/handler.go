package devices

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/saitadikonda99/deployOS/internal/auth"
	"github.com/saitadikonda99/deployOS/pkg/api"
	"github.com/saitadikonda99/deployOS/pkg/protocol"
)

// Handler adapts Service to HTTP: parsing requests, authenticating
// callers, and translating Service errors into status codes.
type Handler struct {
	service       *Service
	authenticator auth.Authenticator
	logger        *slog.Logger
}

// NewHandler builds a Handler.
func NewHandler(service *Service, authenticator auth.Authenticator, logger *slog.Logger) *Handler {
	return &Handler{service: service, authenticator: authenticator, logger: logger}
}

// Register handles POST /api/v1/devices/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	user, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	var req protocol.DeviceRegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.service.Register(r.Context(), user.ID, RegisterInput{
		DeviceID:        req.DeviceID,
		Hostname:        req.Hostname,
		OperatingSystem: req.OperatingSystem,
		Architecture:    req.Architecture,
		CPUCores:        req.CPUCores,
		MemoryBytes:     req.MemoryBytes,
		DeployOSVersion: req.DeployOSVersion,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, protocol.DeviceRegisterResponse{
		DeviceID:  result.Device.ID,
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
	})
}

// List handles GET /api/v1/devices.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user, ok := h.authenticate(w, r)
	if !ok {
		return
	}

	deviceList, err := h.service.List(r.Context(), user.ID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	resp := api.ListDevicesResponse{Devices: make([]api.Device, 0, len(deviceList))}
	for _, d := range deviceList {
		resp.Devices = append(resp.Devices, api.Device{
			ID:              d.ID.String(),
			Hostname:        d.Hostname,
			OperatingSystem: d.OperatingSystem,
			Architecture:    d.Architecture,
			Status:          d.Status,
			CreatedAt:       d.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) authenticate(w http.ResponseWriter, r *http.Request) (auth.User, bool) {
	token, ok := auth.BearerToken(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return auth.User{}, false
	}

	user, err := h.authenticator.Authenticate(r.Context(), token)
	if err != nil {
		if !errors.Is(err, auth.ErrInvalidToken) {
			h.logger.Error("authenticating request", slog.Any("error", err))
		}
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return auth.User{}, false
	}

	return user, true
}

func (h *Handler) writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrOwnedByAnotherUser):
		writeError(w, http.StatusConflict, "device is already registered to another account")
	default:
		h.logger.Error("device request failed", slog.Any("error", err))
		writeError(w, http.StatusInternalServerError, "internal error")
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
