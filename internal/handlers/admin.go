package handlers

import (
	"net/http"
	"time"

	"github.com/wanjiru-noreen/halisi/internal/services"
)

func (h *Resources) adminDashboard(_ http.ResponseWriter, r *http.Request, _ int64) (any, error) {
	period, err := services.ParseDashboardPeriod(r.URL.Query(), time.Now())
	if err != nil {
		return nil, &resourceValidation{message: err.Error()}
	}
	return (&services.AdminDashboard{Pool: h.pool}).Load(r.Context(), period)
}
