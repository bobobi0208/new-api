package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const (
	channelLoadDefaultWindowMinutes = 5
	channelLoadMaxWindowMinutes     = 1440
)

// channelLoadRow is one channel's load snapshot in the overview.
//
// inflight is a per-node live gauge (requests currently executing against the
// channel on the serving instance). requests/errors/rpm/error_rate come from the
// shared log DB; failover_entries from the shared affinity failure cache.
type channelLoadRow struct {
	ChannelId       int     `json:"channel_id"`
	ChannelName     string  `json:"channel_name"`
	Status          int     `json:"status"`
	Inflight        int64   `json:"inflight"`
	Requests        int64   `json:"requests"`
	Errors          int64   `json:"errors"`
	Rpm             float64 `json:"rpm"`
	ErrorRate       float64 `json:"error_rate"`
	FailoverEntries int     `json:"failover_entries"`
}

// GetChannelLoadOverview returns a per-channel load snapshot so an operator can
// decide whether to recover affinity / reschedule. AdminAuth-gated by its route.
func GetChannelLoadOverview(c *gin.Context) {
	windowMinutes := channelLoadDefaultWindowMinutes
	if raw := c.Query("window_minutes"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			windowMinutes = v
		}
	}
	if windowMinutes < 1 {
		windowMinutes = 1
	}
	if windowMinutes > channelLoadMaxWindowMinutes {
		windowMinutes = channelLoadMaxWindowMinutes
	}
	sinceTs := time.Now().Add(-time.Duration(windowMinutes) * time.Minute).Unix()

	inflight := service.SnapshotChannelInflight()

	reqStats, err := model.GetChannelRequestStats(sinceTs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	failoverEntries, err := service.CountChannelAffinityFailureEntriesByChannel()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	channels, err := model.GetChannelLiteList()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	names := make(map[int]string, len(channels))
	statuses := make(map[int]int, len(channels))
	ids := make(map[int]struct{}, len(channels))
	for _, ch := range channels {
		names[ch.Id] = ch.Name
		statuses[ch.Id] = ch.Status
		ids[ch.Id] = struct{}{}
	}
	// Include channels that appear only in a metric source (e.g. a just-deleted
	// channel still has in-flight requests draining, or recent logs/trips).
	for id := range inflight {
		ids[id] = struct{}{}
	}
	for id := range reqStats {
		ids[id] = struct{}{}
	}
	for id := range failoverEntries {
		ids[id] = struct{}{}
	}

	rows := make([]channelLoadRow, 0, len(ids))
	for id := range ids {
		stat := reqStats[id]
		errorRate := 0.0
		if total := stat.Requests + stat.Errors; total > 0 {
			errorRate = float64(stat.Errors) / float64(total)
		}
		name, ok := names[id]
		if !ok {
			name = "#" + strconv.Itoa(id)
		}
		rows = append(rows, channelLoadRow{
			ChannelId:       id,
			ChannelName:     name,
			Status:          statuses[id],
			Inflight:        inflight[id],
			Requests:        stat.Requests,
			Errors:          stat.Errors,
			Rpm:             float64(stat.Requests) / float64(windowMinutes),
			ErrorRate:       errorRate,
			FailoverEntries: failoverEntries[id],
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"window_minutes":    windowMinutes,
			"error_log_enabled": constant.ErrorLogEnabled,
			"channels":          rows,
		},
	})
}
