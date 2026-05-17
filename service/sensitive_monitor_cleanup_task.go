package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/bytedance/gopkg/util/gopool"
)

const (
	sensitiveMonitorHitRetention       = 24 * time.Hour
	sensitiveMonitorHitCleanupInterval = time.Hour
)

var (
	sensitiveMonitorHitCleanupOnce    sync.Once
	sensitiveMonitorHitCleanupRunning atomic.Bool
)

func StartSensitiveMonitorHitCleanupTask() {
	sensitiveMonitorHitCleanupOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		gopool.Go(func() {
			logger.LogInfo(context.Background(), fmt.Sprintf("sensitive monitor hit cleanup task started: interval=%s retention=%s", sensitiveMonitorHitCleanupInterval, sensitiveMonitorHitRetention))
			ticker := time.NewTicker(sensitiveMonitorHitCleanupInterval)
			defer ticker.Stop()
			for range ticker.C {
				runSensitiveMonitorHitCleanupOnce()
			}
		})
	})
}

func runSensitiveMonitorHitCleanupOnce() {
	if !sensitiveMonitorHitCleanupRunning.CompareAndSwap(false, true) {
		return
	}
	defer sensitiveMonitorHitCleanupRunning.Store(false)

	cutoff := time.Now().Add(-sensitiveMonitorHitRetention)
	deleted, err := model.ClearSensitiveWordHitsBefore(cutoff)
	if err != nil {
		logger.LogWarn(context.Background(), fmt.Sprintf("sensitive monitor hit cleanup failed: %v", err))
		return
	}
	if deleted > 0 {
		logger.LogInfo(context.Background(), fmt.Sprintf("sensitive monitor hit cleanup: deleted=%d", deleted))
	}
}
