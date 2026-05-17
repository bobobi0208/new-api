package reconciliation

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
)

const (
	dailyTaskCheckInterval = 5 * time.Minute
)

var (
	dailyTaskOnce            sync.Once
	dailyTaskRunning         atomic.Bool
	dailyTaskLastExecutedDay atomic.Int64 // 存 YYYYMMDD 整数防止同一天多跑
)

// StartDailyTask 启动"日终增量对账"定时任务。
// 每 5 分钟检查一次，若到达 ReconciliationSetting.DailyJobHour 且当天还没跑过，
// 就按昨日 00:00–24:00 的窗口对所有启用渠道做一次对账，并清理 RetainDays 之前的旧记录。
func StartDailyTask() {
	dailyTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), "reconciliation daily task started")
			ticker := time.NewTicker(dailyTaskCheckInterval)
			defer ticker.Stop()
			tickDaily()
			for range ticker.C {
				tickDaily()
			}
		})
	})
}

func tickDaily() {
	setting := operation_setting.GetReconciliationSetting()
	if !setting.Enabled {
		return
	}
	now := time.Now()
	if now.Hour() != setting.DailyJobHour {
		return
	}
	today := int64(now.Year()*10000 + int(now.Month())*100 + now.Day())
	if dailyTaskLastExecutedDay.Load() == today {
		return
	}
	if !dailyTaskRunning.CompareAndSwap(false, true) {
		return
	}
	defer dailyTaskRunning.Store(false)
	dailyTaskLastExecutedDay.Store(today)

	runDailyOnce(now)
}

func runDailyOnce(now time.Time) {
	ctx := context.Background()
	setting := operation_setting.GetReconciliationSetting()

	// 昨日 00:00 → 今日 00:00（本机时区）
	yStart := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	yEnd := yStart.AddDate(0, 0, 1)

	configs, err := model.ListEnabledReconciliationChannelConfigs()
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("reconciliation daily: list configs failed: %v", err))
		return
	}
	matched, mismatched, errored := 0, 0, 0
	for _, cfg := range configs {
		rec, err := RunOnce(ctx, RunOptions{
			ChannelId:   cfg.ChannelId,
			RunType:     model.ReconciliationRunTypeDaily,
			WindowStart: yStart,
			WindowEnd:   yEnd,
		})
		if err != nil {
			errored++
			logger.LogWarn(ctx, fmt.Sprintf("reconciliation daily: channel %d failed: %v", cfg.ChannelId, err))
			continue
		}
		switch rec.Status {
		case model.ReconciliationStatusMatch:
			matched++
		case model.ReconciliationStatusMismatch:
			mismatched++
		case model.ReconciliationStatusError:
			errored++
		}
	}

	// 历史记录清理（按 RetainDays）
	if setting.RetainDays > 0 {
		cutoff := now.AddDate(0, 0, -setting.RetainDays)
		if deleted, derr := model.DeleteReconciliationRecordsBefore(cutoff); derr != nil {
			logger.LogWarn(ctx, fmt.Sprintf("reconciliation daily: cleanup failed: %v", derr))
		} else if deleted > 0 {
			logger.LogInfo(ctx, fmt.Sprintf("reconciliation daily: cleaned %d old records before %s", deleted, cutoff.Format(time.RFC3339)))
		}
	}

	logger.LogInfo(ctx, fmt.Sprintf(
		"reconciliation daily: window=[%s, %s) scanned=%d match=%d mismatch=%d error=%d",
		yStart.Format("2006-01-02"), yEnd.Format("2006-01-02"),
		len(configs), matched, mismatched, errored,
	))
}
