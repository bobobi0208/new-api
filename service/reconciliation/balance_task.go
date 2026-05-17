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

var (
	balanceTaskOnce    sync.Once
	balanceTaskRunning atomic.Bool
)

// StartBalanceTask 启动"余额哨兵"定时任务。
// 每轮按 ReconciliationSetting.BalancePollIntervalSec 间隔，遍历所有启用的对账渠道，
// 拉一次上游 used/remain 快照，与本地累计 UsedQuota 比对。
func StartBalanceTask() {
	balanceTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), "reconciliation balance sentry task started")
			// 初始等待一个短延迟，让其他启动期任务先就绪
			time.Sleep(30 * time.Second)
			for {
				setting := operation_setting.GetReconciliationSetting()
				interval := time.Duration(setting.SanitizedBalancePollInterval()) * time.Second
				if setting.Enabled {
					runBalanceOnce()
				}
				time.Sleep(interval)
			}
		})
	})
}

func runBalanceOnce() {
	if !balanceTaskRunning.CompareAndSwap(false, true) {
		return
	}
	defer balanceTaskRunning.Store(false)

	ctx := context.Background()
	configs, err := model.ListEnabledReconciliationChannelConfigs()
	if err != nil {
		logger.LogError(ctx, fmt.Sprintf("reconciliation balance: list configs failed: %v", err))
		return
	}
	if len(configs) == 0 {
		return
	}
	matched, mismatched, errored := 0, 0, 0
	for _, cfg := range configs {
		rec, err := RunOnce(ctx, RunOptions{
			ChannelId: cfg.ChannelId,
			RunType:   model.ReconciliationRunTypeBalance,
		})
		if err != nil {
			errored++
			logger.LogWarn(ctx, fmt.Sprintf("reconciliation balance: channel %d run failed: %v", cfg.ChannelId, err))
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
	logger.LogInfo(ctx,
		fmt.Sprintf("reconciliation balance: scanned=%d match=%d mismatch=%d error=%d",
			len(configs), matched, mismatched, errored),
	)
}
