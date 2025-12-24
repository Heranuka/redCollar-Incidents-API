package domain

type IncidentStats struct {
	UserCount   int64 `json:"unique_users"`
	TotalChecks int64 `json:"total_checks"`
	Minutes     int   `json:"minutes"`
}

type StatsRequest struct {
	Minutes int `query:"minutes" validate:"min=1,max=1440"` // 1 день max
}
