package dto

type GroupAvailabilitySnapshot struct {
	Group         string   `json:"group"`
	SuccessRate   *float64 `json:"success_rate"`
	OnlineRate    float64  `json:"online_rate"`
	AvgLatencyMs  int      `json:"avg_latency_ms"`
	RequestCount  int64    `json:"request_count"`
	ErrorCount    int64    `json:"error_count"`
	ChannelTotal  int      `json:"channel_total"`
	ChannelOnline int      `json:"channel_online"`
}

type GroupAvailabilityOverview struct {
	Groups        []GroupAvailabilitySnapshot `json:"groups"`
	WindowSeconds int64                       `json:"window_seconds"`
	GeneratedAt   int64                       `json:"generated_at"`
}

type AvailabilityBucket struct {
	Bucket       int64 `json:"bucket"`
	SuccessCount int64 `json:"success_count"`
	ErrorCount   int64 `json:"error_count"`
	AvgLatencyMs int   `json:"avg_latency_ms"`
}

type ChannelAvailabilityRow struct {
	ChannelId    int    `json:"channel_id"`
	Name         string `json:"name"`
	Status       int    `json:"status"`
	ResponseTime int    `json:"response_time_ms"`
	TestTime     int64  `json:"test_time"`
	RequestCount int64  `json:"request_count"`
	ErrorCount   int64  `json:"error_count"`
}

type GroupAvailabilityTimeseries struct {
	Group         string                   `json:"group"`
	Range         string                   `json:"range"`
	BucketSec     int64                    `json:"bucket_sec"`
	StartSec      int64                    `json:"start_sec"`
	EndSec        int64                    `json:"end_sec"`
	Points        []AvailabilityBucket     `json:"points"`
	P50Ms         *int                     `json:"p50_ms"`
	P95Ms         *int                     `json:"p95_ms"`
	SuccessRate   *float64                 `json:"success_rate"`
	RequestCount  int64                    `json:"request_count"`
	ErrorCount    int64                    `json:"error_count"`
	AvgLatencyMs  int                      `json:"avg_latency_ms"`
	ChannelTotal  int                      `json:"channel_total"`
	ChannelOnline int                      `json:"channel_online"`
	Channels      []ChannelAvailabilityRow `json:"channels"`
}
