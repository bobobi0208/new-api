package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// ReconciliationSetting 控制"上游对账"子系统的运行参数。
// 通过 config.GlobalConfig.Register 接入 OptionMap 热更新机制，前端 admin 设置即时生效。
type ReconciliationSetting struct {
	// 总开关；关闭时余额哨兵和日终任务都不会执行。
	Enabled bool `json:"enabled"`
	// 余额哨兵任务的轮询间隔（秒）。最小 60 秒，避免对上游产生过高压力。
	BalancePollIntervalSec int `json:"balance_poll_interval_sec"`
	// 日终增量对账任务执行的本机时区小时数（0-23）。
	DailyJobHour int `json:"daily_job_hour"`
	// 绝对偏差阈值（USD）。 |upstream_used - local_used| 超过此值即标 mismatch。
	AbsThresholdUSD float64 `json:"abs_threshold_usd"`
	// 相对偏差阈值。 |delta| / upstream_used 超过此值即标 mismatch。
	RelThreshold float64 `json:"rel_threshold"`
	// 上游 HTTP 请求超时（秒）。
	HttpTimeoutSec int `json:"http_timeout_sec"`
	// 历史记录保留天数；超过的记录由日终任务批量清理。
	RetainDays int `json:"retain_days"`
}

var reconciliationSetting = ReconciliationSetting{
	Enabled:                true,
	BalancePollIntervalSec: 600,
	DailyJobHour:           2,
	AbsThresholdUSD:        0.01,
	RelThreshold:           0.05,
	HttpTimeoutSec:         15,
	RetainDays:             90,
}

func init() {
	config.GlobalConfig.Register("reconciliation_setting", &reconciliationSetting)
}

func GetReconciliationSetting() *ReconciliationSetting {
	return &reconciliationSetting
}

// SanitizedBalancePollInterval 返回归一化后的余额哨兵间隔（防呆，下限 60 秒）。
func (s *ReconciliationSetting) SanitizedBalancePollInterval() int {
	if s.BalancePollIntervalSec < 60 {
		return 60
	}
	return s.BalancePollIntervalSec
}

// SanitizedHttpTimeout 返回归一化后的 HTTP 超时（下限 3 秒，上限 120 秒）。
func (s *ReconciliationSetting) SanitizedHttpTimeout() int {
	if s.HttpTimeoutSec < 3 {
		return 3
	}
	if s.HttpTimeoutSec > 120 {
		return 120
	}
	return s.HttpTimeoutSec
}
