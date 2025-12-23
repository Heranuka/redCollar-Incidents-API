package domain

import (
	"time"

	"github.com/google/uuid"
)

type WebhookPayload struct {
	UserID    string      `json:"user_id"`
	Lat       float64     `json:"lat"`
	Lng       float64     `json:"lng"`
	Incidents []uuid.UUID `json:"incidents"`
	CheckedAt time.Time   `json:"checked_at"`
}
