package domain

type CreateIncidentRequest struct {
	Lat      float64        `json:"lat" validate:"required,lat"`
	Lng      float64        `json:"lng" validate:"required,lng"`
	RadiusKM float64        `json:"radius_km" validate:"required,min=0.1,max=100"`
	Status   IncidentStatus `json:"status" validate:"omitempty,oneof=active inactive"`
}

type UpdateIncidentRequest struct {
	Lat      *float64        `json:"lat" validate:"omitempty,lat"`
	Lng      *float64        `json:"lng" validate:"omitempty,lng"`
	RadiusKM *float64        `json:"radius_km" validate:"omitempty,min=0.1,max=100"`
	Status   *IncidentStatus `json:"status" validate:"omitempty,oneof=active inactive"`
}

type ListIncidentsRequest struct {
	Page  int `query:"page" validate:"min=1"`
	Limit int `query:"limit" validate:"min=1,max=100"`
}

type ListIncidentsResponse struct {
	Incidents []Incident `json:"incidents"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
	Total     int64      `json:"total"`
}
