package domain

import (
	"time"

	"github.com/google/uuid"
)

type IncidentStatus string

const (
	IncidentActive   IncidentStatus = "active"
	IncidentInactive IncidentStatus = "inactive"
)

type Incident struct {
	ID        uuid.UUID      `json:"id"`
	Lat       float64        `json:"lat" validate:"required,lat"` // -90..90
	Lng       float64        `json:"lng" validate:"required,lng"` // -180..180
	RadiusKM  float64        `json:"radius_km" validate:"required,min=0.1,max=100"`
	Status    IncidentStatus `json:"status"`
	CreatedAt time.Time      `json:"created_at"`
}
