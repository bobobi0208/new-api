package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetChannelRequestStats(t *testing.T) {
	now := time.Now().Unix()
	inWindow := now - 60
	outOfWindow := now - 3600

	// Unique high channel ids to avoid colliding with other tests' rows.
	const chA = 970001
	const chB = 970002

	seed := []Log{
		// chA: 3 consume + 1 error in window
		{ChannelId: chA, Type: LogTypeConsume, CreatedAt: inWindow},
		{ChannelId: chA, Type: LogTypeConsume, CreatedAt: inWindow},
		{ChannelId: chA, Type: LogTypeConsume, CreatedAt: inWindow},
		{ChannelId: chA, Type: LogTypeError, CreatedAt: inWindow},
		// chA: out-of-window rows must be excluded
		{ChannelId: chA, Type: LogTypeConsume, CreatedAt: outOfWindow},
		{ChannelId: chA, Type: LogTypeError, CreatedAt: outOfWindow},
		// chB: 1 consume in window
		{ChannelId: chB, Type: LogTypeConsume, CreatedAt: inWindow},
		// noise: a non-counted type in window must be ignored
		{ChannelId: chA, Type: LogTypeManage, CreatedAt: inWindow},
	}
	require.NoError(t, LOG_DB.Create(&seed).Error)
	t.Cleanup(func() {
		LOG_DB.Where("channel_id IN ?", []int{chA, chB}).Delete(&Log{})
	})

	stats, err := GetChannelRequestStats(now - 300)
	require.NoError(t, err)

	require.Equal(t, ChannelReqStat{Requests: 3, Errors: 1}, stats[chA])
	require.Equal(t, ChannelReqStat{Requests: 1, Errors: 0}, stats[chB])
}
