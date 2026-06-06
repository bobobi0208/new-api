package service

import (
	"sync"
	"sync/atomic"
)

// channelInflight tracks the number of upstream relay attempts currently
// executing against each channel on THIS node. It is an in-memory, per-node
// gauge: the relay loop is a hot path, so a Redis round-trip per attempt is
// unacceptable, and per-node counters are leak-safe (a process restart resets
// them to zero, whereas a crashed Redis INCR would never be DECR'd).
//
// The map is keyed by channel id and holds *int64 counters mutated atomically.
// The pointer for a given channel is allocated once via LoadOrStore and never
// replaced, so atomic ops on it are always well-defined.
var channelInflight sync.Map // map[int]*int64

func channelInflightCounter(channelID int) *int64 {
	if v, ok := channelInflight.Load(channelID); ok {
		return v.(*int64)
	}
	v, _ := channelInflight.LoadOrStore(channelID, new(int64))
	return v.(*int64)
}

// IncChannelInflight increments the live in-flight gauge for a channel.
func IncChannelInflight(channelID int) {
	if channelID <= 0 {
		return
	}
	atomic.AddInt64(channelInflightCounter(channelID), 1)
}

// DecChannelInflight decrements the live in-flight gauge for a channel. It is
// safe to call on every exit path; the counter is clamped at zero so a single
// mis-bracketed Dec can never drive the gauge permanently negative.
func DecChannelInflight(channelID int) {
	if channelID <= 0 {
		return
	}
	p := channelInflightCounter(channelID)
	if atomic.AddInt64(p, -1) < 0 {
		atomic.StoreInt64(p, 0)
	}
}

// SnapshotChannelInflight returns a point-in-time copy of the per-channel
// in-flight counts. Channels whose counter has decayed to zero are omitted.
func SnapshotChannelInflight() map[int]int64 {
	out := make(map[int]int64)
	channelInflight.Range(func(key, value any) bool {
		id, ok := key.(int)
		if !ok {
			return true
		}
		p, ok := value.(*int64)
		if !ok {
			return true
		}
		if n := atomic.LoadInt64(p); n > 0 {
			out[id] = n
		}
		return true
	})
	return out
}
