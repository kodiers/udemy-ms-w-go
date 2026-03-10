package http

import (
	"encoding/json"
	"log"
	"net/http"
	"ride-sharing/services/trip-service/internal/domain"
	"ride-sharing/services/trip-service/internal/service"
	"ride-sharing/shared/types"
)

type HttpHandler struct {
	Svc *service.TripService
}

type previewTripRequest struct {
	UserId      string           `json:"userID"`
	Pickup      types.Coordinate `json:"pickup"`
	Destination types.Coordinate `json:"destination"`
}

func (handler *HttpHandler) HandleTripPreview(w http.ResponseWriter, r *http.Request) {
	var reqBody previewTripRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	fare := &domain.RideFareModel{
		UserID: "42",
	}
	ctx := r.Context()
	t, err := handler.Svc.CreateTrip(ctx, fare)
	if err != nil {
		log.Println(err)
	}

	writeJson(w, http.StatusOK, t)
}

func writeJson(w http.ResponseWriter, status int, data any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}
