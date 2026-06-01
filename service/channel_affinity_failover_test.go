package service

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func buildFailoverContextForTest(suffix string, threshold, windowSeconds int) *gin.Context {
	rec := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(rec)
	setChannelAffinityContext(ctx, channelAffinityMeta{
		CacheKey:             "test:" + suffix,
		CacheKeySuffix:       suffix,
		TTLSeconds:           600,
		FailoverEnabled:      true,
		FailoverThreshold:    threshold,
		FailoverWindowSecond: windowSeconds,
	})
	return ctx
}

func TestIsChannelSideFailure(t *testing.T) {
	// Channel-side: 5xx retryable → counts.
	require.True(t, isChannelSideFailure(
		types.NewErrorWithStatusCode(errors.New("boom"), "server_error", 500)))
	// Channel error code prefix → counts.
	require.True(t, isChannelSideFailure(
		types.NewError(errors.New("down"), types.ErrorCode("channel:unavailable"))))
	// Network-level (no HTTP status) → counts.
	require.True(t, isChannelSideFailure(
		types.NewError(errors.New("dial tcp refused"), "connection_error")))

	// User-side 400 (not in retry ranges) → does not count.
	require.False(t, isChannelSideFailure(
		types.NewErrorWithStatusCode(errors.New("bad"), "invalid_request", 400)))
	// Skip-retry flagged → does not count.
	require.False(t, isChannelSideFailure(
		types.NewErrorWithStatusCode(errors.New("sensitive"), "sensitive_words_detected", 400, types.ErrOptionWithSkipRetry())))
	// 2xx → does not count.
	require.False(t, isChannelSideFailure(
		types.NewErrorWithStatusCode(errors.New("ok"), "ok", 200)))
	// nil → does not count.
	require.False(t, isChannelSideFailure(nil))
}

func TestChannelAffinityFailover_AccumulatesAndTrips(t *testing.T) {
	suffix := fmt.Sprintf("fo_%d", time.Now().UnixNano())
	channelID := 7
	threshold := 3
	ctx := buildFailoverContextForTest(suffix, threshold, 600)

	chanErr := types.NewErrorWithStatusCode(errors.New("boom"), "server_error", 500)

	for i := 1; i < threshold; i++ {
		RecordChannelAffinityFailure(ctx, channelID, chanErr)
		require.False(t, channelAffinityChannelTripped(suffix, channelID, threshold),
			"should not trip before threshold (i=%d)", i)
	}
	RecordChannelAffinityFailure(ctx, channelID, chanErr)
	require.True(t, channelAffinityChannelTripped(suffix, channelID, threshold),
		"should trip at threshold")

	// A different channel under the same affinity key is unaffected.
	require.False(t, channelAffinityChannelTripped(suffix, channelID+1, threshold))
}

func TestChannelAffinityFailover_UserSideDoesNotCount(t *testing.T) {
	suffix := fmt.Sprintf("fo_%d", time.Now().UnixNano())
	channelID := 9
	threshold := 2
	ctx := buildFailoverContextForTest(suffix, threshold, 600)

	userErr := types.NewErrorWithStatusCode(errors.New("bad"), "invalid_request", 400)
	for i := 0; i < 5; i++ {
		RecordChannelAffinityFailure(ctx, channelID, userErr)
	}
	require.False(t, channelAffinityChannelTripped(suffix, channelID, threshold),
		"user-side 4xx must not accumulate")
}

func TestChannelAffinityFailover_WindowExpiryResets(t *testing.T) {
	suffix := fmt.Sprintf("fo_%d", time.Now().UnixNano())
	channelID := 11
	threshold := 2
	// 1-second window so the TTL elapses quickly.
	ctx := buildFailoverContextForTest(suffix, threshold, 1)

	chanErr := types.NewErrorWithStatusCode(errors.New("boom"), "server_error", 500)
	RecordChannelAffinityFailure(ctx, channelID, chanErr)
	RecordChannelAffinityFailure(ctx, channelID, chanErr)
	require.True(t, channelAffinityChannelTripped(suffix, channelID, threshold))

	time.Sleep(1500 * time.Millisecond)
	require.False(t, channelAffinityChannelTripped(suffix, channelID, threshold),
		"counter must reset after the window TTL elapses")
}

func TestClearChannelAffinityFailuresByChannel(t *testing.T) {
	stamp := time.Now().UnixNano()
	suffixA := fmt.Sprintf("clrA_%d", stamp)
	suffixB := fmt.Sprintf("clrB_%d", stamp)
	suffixC := fmt.Sprintf("clrC_%d", stamp)

	cache := getChannelAffinityFailureCache()
	ttl := 600 * time.Second
	require.NoError(t, cache.SetWithTTL(channelAffinityFailureKey(suffixA, 5), 3, ttl))
	require.NoError(t, cache.SetWithTTL(channelAffinityFailureKey(suffixB, 5), 2, ttl))
	require.NoError(t, cache.SetWithTTL(channelAffinityFailureKey(suffixC, 7), 4, ttl))

	deleted, err := ClearChannelAffinityFailuresByChannel(5)
	require.NoError(t, err)
	require.Equal(t, 2, deleted, "both ch:5 counters cleared")

	_, found, err := cache.Get(channelAffinityFailureKey(suffixA, 5))
	require.NoError(t, err)
	require.False(t, found, "ch:5 / suffixA must be gone")
	_, found, err = cache.Get(channelAffinityFailureKey(suffixB, 5))
	require.NoError(t, err)
	require.False(t, found, "ch:5 / suffixB must be gone")

	count, found, err := cache.Get(channelAffinityFailureKey(suffixC, 7))
	require.NoError(t, err)
	require.True(t, found, "ch:7 must be untouched")
	require.Equal(t, 4, count)

	// Invalid id is rejected.
	_, err = ClearChannelAffinityFailuresByChannel(0)
	require.Error(t, err)
}

// TestChannelAffinity_TripKeepsAffinityAndSuppresses drives the real selection
// path: once a channel trips, GetPreferredChannelByAffinity must report a miss,
// keep the affinity entry (no longer deletes it), and flag the request as
// suppressed so the fallback success does not drift the affinity away.
func TestChannelAffinity_TripKeepsAffinityAndSuppresses(t *testing.T) {
	gin.SetMode(gin.TestMode)
	model := "gpt-test"
	group := "grp"
	promptKey := fmt.Sprintf("pck_%d", time.Now().UnixNano())
	body := fmt.Sprintf(`{"prompt_cache_key":%q}`, promptKey)

	newCtx := func() *gin.Context {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", strings.NewReader(body))
		ctx.Request.Header.Set("Content-Type", "application/json")
		ctx.Set(common.KeyRequestBody, []byte(body))
		return ctx
	}

	// First pass: no affinity yet → miss, but context/meta is populated.
	ctx1 := newCtx()
	_, found := GetPreferredChannelByAffinity(ctx1, model, group)
	require.False(t, found, "no affinity entry on first pass")
	meta, ok := getChannelAffinityMeta(ctx1)
	require.True(t, ok)
	suffix := meta.CacheKeySuffix
	require.NotEmpty(t, suffix)

	const preferredChannel = 42
	// Seed the affinity at the same key the selector reads.
	require.NoError(t, getChannelAffinityCache().SetWithTTL(suffix, preferredChannel, 600*time.Second))

	// Accumulate channel-side failures to the trip threshold.
	chanErr := types.NewErrorWithStatusCode(errors.New("boom"), "server_error", 500)
	for i := 0; i < 3; i++ {
		RecordChannelAffinityFailure(ctx1, preferredChannel, chanErr)
	}
	require.True(t, channelAffinityChannelTripped(suffix, preferredChannel, 3))

	// Second pass: tripped → miss + suppressed flag, affinity entry preserved.
	ctx2 := newCtx()
	_, found = GetPreferredChannelByAffinity(ctx2, model, group)
	require.False(t, found, "tripped channel must not be preferred")
	require.True(t, ctx2.GetBool(ginKeyChannelAffinitySuppressed), "request must be marked suppressed")

	stored, present, err := getChannelAffinityCache().Get(suffix)
	require.NoError(t, err)
	require.True(t, present, "affinity entry must NOT be deleted on trip")
	require.Equal(t, preferredChannel, stored)
}

// TestRecordChannelAffinity_SuppressedPreventsDrift verifies the suppressed
// guard: a request that fell back to a backup channel must not overwrite the
// original affinity.
func TestRecordChannelAffinity_SuppressedPreventsDrift(t *testing.T) {
	gin.SetMode(gin.TestMode)
	const original = 42
	const backup = 99

	key := fmt.Sprintf("new-api:drift:%d", time.Now().UnixNano())
	seed := func() {
		require.NoError(t, getChannelAffinityCache().SetWithTTL(key, original, 600*time.Second))
	}
	makeCtx := func() *gin.Context {
		ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
		setChannelAffinityContext(ctx, channelAffinityMeta{CacheKey: key, TTLSeconds: 600})
		return ctx
	}

	// Suppressed → original affinity is preserved.
	seed()
	ctxSuppressed := makeCtx()
	ctxSuppressed.Set(ginKeyChannelAffinitySuppressed, true)
	RecordChannelAffinity(ctxSuppressed, backup)
	v, found, err := getChannelAffinityCache().Get(key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, original, v, "suppressed request must not drift affinity")

	// Control: without the flag the affinity is overwritten, proving the guard
	// is responsible for the difference above.
	seed()
	ctxNormal := makeCtx()
	RecordChannelAffinity(ctxNormal, backup)
	v, found, err = getChannelAffinityCache().Get(key)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, backup, v, "normal request should record the new channel")
}
