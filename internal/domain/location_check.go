package domain

import (
	"time"

	"github.com/google/uuid"
)

type LocationCheckRequest struct {
	UserID string  `json:"user_id" validate:"required,uuid"`
	Lat    float64 `json:"lat" validate:"required,lat"`
	Lng    float64 `json:"lng" validate:"required,lng"`
}

type LocationCheckResponse struct {
	Incidents []string `json:"incidents"` // массив uuid строк
}

type LocationCheck struct {
	ID          uuid.UUID   `json:"id"`
	UserID      uuid.UUID   `json:"user_id"`
	Lat         float64     `json:"lat"`
	Lng         float64     `json:"lng"`
	IncidentIDs []uuid.UUID `json:"incident_ids"`
	CheckedAt   time.Time   `json:"checked_at"`
}
