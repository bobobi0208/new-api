package service

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChannelInflight_IncDecSnapshot(t *testing.T) {
	const ch = 90001
	require.Equal(t, int64(0), SnapshotChannelInflight()[ch])

	IncChannelInflight(ch)
	IncChannelInflight(ch)
	IncChannelInflight(ch)
	require.Equal(t, int64(3), SnapshotChannelInflight()[ch])

	DecChannelInflight(ch)
	require.Equal(t, int64(2), SnapshotChannelInflight()[ch])

	// Decays to zero -> omitted from snapshot.
	DecChannelInflight(ch)
	DecChannelInflight(ch)
	_, present := SnapshotChannelInflight()[ch]
	require.False(t, present)
}

func TestChannelInflight_ClampsAtZero(t *testing.T) {
	const ch = 90002
	// Dec on a never-seen / already-zero channel must not go negative.
	DecChannelInflight(ch)
	DecChannelInflight(ch)
	_, present := SnapshotChannelInflight()[ch]
	require.False(t, present)

	IncChannelInflight(ch)
	require.Equal(t, int64(1), SnapshotChannelInflight()[ch])
}

func TestChannelInflight_IgnoresNonPositiveIDs(t *testing.T) {
	IncChannelInflight(0)
	IncChannelInflight(-5)
	DecChannelInflight(0)
	require.Equal(t, int64(0), SnapshotChannelInflight()[0])
	require.Equal(t, int64(0), SnapshotChannelInflight()[-5])
}

func TestChannelInflight_ConcurrentBalanced(t *testing.T) {
	const ch = 90003
	const goroutines = 64
	const iters = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				IncChannelInflight(ch)
				DecChannelInflight(ch)
			}
		}()
	}
	wg.Wait()

	// Every Inc is balanced by a Dec, so nothing should remain.
	_, present := SnapshotChannelInflight()[ch]
	require.False(t, present)
}
