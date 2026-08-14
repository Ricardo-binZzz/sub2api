package service

import "time"

// UpstreamPerformanceMetric is a read-only latency summary for one routing dimension.
type UpstreamPerformanceMetric struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Platform        string   `json:"platform,omitempty"`
	Requests        int64    `json:"requests"`
	Successful      int64    `json:"successful"`
	SuccessRate     float64  `json:"success_rate"`
	FirstTokenP50Ms *float64 `json:"first_token_p50_ms,omitempty"`
	FirstTokenP95Ms *float64 `json:"first_token_p95_ms,omitempty"`
	DurationP50Ms   *float64 `json:"duration_p50_ms,omitempty"`
	DurationP95Ms   *float64 `json:"duration_p95_ms,omitempty"`
}

type UpstreamPerformanceReport struct {
	StartTime   time.Time                     `json:"start_time"`
	EndTime     time.Time                     `json:"end_time"`
	GeneratedAt time.Time                     `json:"generated_at"`
	Groups      []UpstreamPerformanceMetric   `json:"groups"`
	Accounts    []UpstreamPerformanceMetric   `json:"accounts"`
	Thresholds  UpstreamPerformanceThresholds `json:"thresholds"`
}

type UpstreamPerformanceThresholds struct {
	FirstTokenP50WarningMs float64 `json:"first_token_p50_warning_ms"`
	FirstTokenP95WarningMs float64 `json:"first_token_p95_warning_ms"`
	SuccessRateWarning     float64 `json:"success_rate_warning"`
}
