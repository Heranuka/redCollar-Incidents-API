package domain

type IncidentStats struct {
	UserCount int64 `json:"user_count"`
}

type StatsRequest struct {
	Minutes int `query:"minutes" validate:"min=1,max=1440"` // 1 день max
}
