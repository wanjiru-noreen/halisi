package handlers

import (
	"net/http"

	"github.com/wanjiru-noreen/halisi/internal/middleware"
	"github.com/wanjiru-noreen/halisi/internal/services"
)

func (h *Resources) getDelivery(_ http.ResponseWriter, r *http.Request, _ int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	user, _ := middleware.CurrentUser(r.Context())
	delivery, err := services.NewDeliveryManagement(h.pool).Get(r.Context(), id, user)
	return map[string]any{"delivery": delivery}, err
}

func (h *Resources) confirmOrder(_ http.ResponseWriter, r *http.Request, owner int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	order, err := services.NewDeliveryManagement(h.pool).Confirm(r.Context(), id, owner)
	return map[string]any{"order": order}, err
}

type assignDeliveryInput struct {
	RiderID int64 `json:"rider_id"`
}

func (h *Resources) assignDelivery(w http.ResponseWriter, r *http.Request, vendorUserID int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	var input assignDeliveryInput
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	delivery, err := services.NewDeliveryManagement(h.pool).Assign(r.Context(), id, vendorUserID, input.RiderID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"delivery": delivery}, nil
}

func (h *Resources) adminAssignDelivery(w http.ResponseWriter, r *http.Request, _ int64) (any, error) {
	id, err := resourceID(r)
	if err != nil {
		return nil, err
	}
	var input assignDeliveryInput
	if !decodeJSON(w, r, &input) {
		return nil, nil
	}
	delivery, err := services.NewDeliveryManagement(h.pool).AdminAssign(r.Context(), id, input.RiderID)
	if err != nil {
		return nil, err
	}
	return map[string]any{"delivery": delivery}, nil
}

func (h *Resources) listApprovedRiders(_ http.ResponseWriter, r *http.Request, _ int64) (any, error) {
	riders, err := services.NewDeliveryManagement(h.pool).ListApprovedRiders(r.Context())
	if err != nil {
		return nil, err
	}
	return map[string]any{"riders": riders}, nil
}
