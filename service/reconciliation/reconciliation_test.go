package reconciliation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestMain(m *testing.M) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		panic("open test db: " + err.Error())
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic("get sql.DB: " + err.Error())
	}
	sqlDB.SetMaxOpenConns(1)
	model.DB = db
	model.LOG_DB = db
	common.UsingSQLite = true
	common.RedisEnabled = false
	common.BatchUpdateEnabled = false
	common.LogConsumeEnabled = true

	if err := db.AutoMigrate(
		&model.Channel{},
		&model.Log{},
		&model.ReconciliationRecord{},
		&model.ReconciliationChannelConfig{},
	); err != nil {
		panic("automigrate: " + err.Error())
	}

	service.InitHttpClient()
	common.RelayTimeout = 30 // 给伪上游留够时间

	os.Exit(m.Run())
}

func resetReconcilTables(t *testing.T) {
	t.Helper()
	require.NoError(t, model.DB.Exec("DELETE FROM reconciliation_records").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM reconciliation_channel_configs").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM logs").Error)
	require.NoError(t, model.DB.Exec("DELETE FROM channels").Error)
}

// startFakeNewApi 起一个模拟 new-api 上游的 httptest server。
// 通过 Authorization Bearer 后缀切换场景，与 tools/reconciliation_e2e/fake_upstream.go 一致。
func startFakeNewApi() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/usage/token/", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if strings.HasPrefix(token, "sk-err") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		if strings.HasPrefix(token, "sk-inconc") {
			_ = json.NewEncoder(w).Encode(map[string]any{"code": true, "message": "ok"})
			return
		}
		used := 500_000.0
		if strings.HasPrefix(token, "sk-mismatch") {
			used = 1_500_000.0
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":    true,
			"message": "ok",
			"data": map[string]any{
				"total_granted":   10_000_000,
				"total_used":      used,
				"total_available": 10_000_000 - used,
				"unlimited_quota": false,
				"expires_at":      0,
			},
		})
	})
	return httptest.NewServer(mux)
}

// startFakeSub2Api 起一个模拟 sub2api 上游的 httptest server。
func startFakeSub2Api() *httptest.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/usage", func(w http.ResponseWriter, r *http.Request) {
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if strings.HasPrefix(token, "sk-err") {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		if strings.HasPrefix(token, "sk-inconc") {
			_ = json.NewEncoder(w).Encode(map[string]any{"mode": "quota_limited"})
			return
		}
		used := 1.0
		if strings.HasPrefix(token, "sk-mismatch") {
			used = 5.0
		}
		hasWindow := r.URL.Query().Get("start_date") != ""
		todayCost := used
		if strings.HasPrefix(token, "sk-mismatch") && hasWindow {
			todayCost = 4.0
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"mode": "quota_limited",
			"quota": map[string]any{
				"limit":     20.0,
				"used":      used,
				"remaining": 20.0 - used,
				"unit":      "USD",
			},
			"usage": map[string]any{
				"today": map[string]any{"cost": todayCost},
				"total": map[string]any{"cost": used},
			},
		})
	})
	return httptest.NewServer(mux)
}

func seedChannel(t *testing.T, id int, key, baseURL string, usedQuotaLocal int64) *model.Channel {
	t.Helper()
	bu := baseURL
	ch := &model.Channel{
		Id:        id,
		Type:      1,
		Key:       key,
		Name:      "test-channel",
		Status:    common.ChannelStatusEnabled,
		BaseURL:   &bu,
		UsedQuota: usedQuotaLocal,
	}
	require.NoError(t, model.DB.Create(ch).Error)
	return ch
}

func seedConfig(t *testing.T, channelId int, upstreamType string) {
	t.Helper()
	cfg := &model.ReconciliationChannelConfig{
		ChannelId:    channelId,
		UpstreamType: upstreamType,
		Enabled:      true,
	}
	require.NoError(t, model.UpsertReconciliationChannelConfig(cfg))
}

func TestRunOnce_NewApi_Match(t *testing.T) {
	resetReconcilTables(t)
	srv := startFakeNewApi()
	defer srv.Close()

	// 本地累计 500_000 quota = $1，上游也报 used=500_000 → delta=0
	seedChannel(t, 101, "sk-match-1", srv.URL, 500_000)
	seedConfig(t, 101, model.ReconciliationUpstreamTypeNewApi)

	rec, err := RunOnce(context.Background(), RunOptions{ChannelId: 101, RunType: model.ReconciliationRunTypeManual})
	require.NoError(t, err)
	require.Equal(t, model.ReconciliationStatusMatch, rec.Status, "msg=%s raw=%s", rec.Message, rec.UpstreamRawJSON)
	require.InDelta(t, 1.0, rec.UpstreamUsedUSD, 1e-9)
	require.InDelta(t, 1.0, rec.LocalUsedUSD, 1e-9)
	require.InDelta(t, 0.0, rec.DeltaUSD, 1e-9)
}

func TestRunOnce_NewApi_Mismatch(t *testing.T) {
	resetReconcilTables(t)
	srv := startFakeNewApi()
	defer srv.Close()

	// 本地 $1，上游 used=1_500_000 = $3 → delta=$2，远超 0.05 相对阈值
	seedChannel(t, 102, "sk-mismatch-1", srv.URL, 500_000)
	seedConfig(t, 102, model.ReconciliationUpstreamTypeNewApi)

	rec, err := RunOnce(context.Background(), RunOptions{ChannelId: 102, RunType: model.ReconciliationRunTypeManual})
	require.NoError(t, err)
	require.Equal(t, model.ReconciliationStatusMismatch, rec.Status)
	require.InDelta(t, 2.0, rec.DeltaUSD, 1e-9)
}

func TestRunOnce_NewApi_MultiKey_Inconclusive(t *testing.T) {
	resetReconcilTables(t)
	srv := startFakeNewApi()
	defer srv.Close()

	// 多 key channel：Adapter 应直接返回 Inconclusive
	bu := srv.URL
	ch := &model.Channel{
		Id: 103, Type: 1, Key: "sk-match-1\nsk-match-2", Name: "multi-key",
		Status: common.ChannelStatusEnabled, BaseURL: &bu, UsedQuota: 500_000,
		ChannelInfo: model.ChannelInfo{IsMultiKey: true},
	}
	require.NoError(t, model.DB.Create(ch).Error)
	seedConfig(t, 103, model.ReconciliationUpstreamTypeNewApi)

	rec, err := RunOnce(context.Background(), RunOptions{ChannelId: 103, RunType: model.ReconciliationRunTypeManual})
	require.NoError(t, err)
	require.Equal(t, model.ReconciliationStatusInconclusive, rec.Status)
	require.Contains(t, rec.Message, "multi-key")
}

func TestRunOnce_NewApi_HttpError(t *testing.T) {
	resetReconcilTables(t)
	srv := startFakeNewApi()
	defer srv.Close()

	seedChannel(t, 104, "sk-err-1", srv.URL, 500_000)
	seedConfig(t, 104, model.ReconciliationUpstreamTypeNewApi)

	rec, err := RunOnce(context.Background(), RunOptions{ChannelId: 104, RunType: model.ReconciliationRunTypeManual})
	require.NoError(t, err)
	// 上游 500 → Adapter 兜底 subscription（404）→ 失败 → 走 error
	require.Equal(t, model.ReconciliationStatusError, rec.Status)
}

func TestRunOnce_Sub2Api_Window_Match(t *testing.T) {
	resetReconcilTables(t)
	srv := startFakeSub2Api()
	defer srv.Close()

	// 本地有一条窗口内 Consume Log，quota=500_000 ($1)，上游窗口聚合 cost=1.0 → match
	now := time.Now()
	wStart := now.Add(-1 * time.Hour)
	wEnd := now.Add(1 * time.Hour)
	seedChannel(t, 201, "sk-match-1", srv.URL, 0)
	seedConfig(t, 201, model.ReconciliationUpstreamTypeSub2Api)
	require.NoError(t, model.DB.Create(&model.Log{
		Type: model.LogTypeConsume, ChannelId: 201, Quota: 500_000,
		CreatedAt: now.Unix(),
	}).Error)

	rec, err := RunOnce(context.Background(), RunOptions{
		ChannelId:   201,
		RunType:     model.ReconciliationRunTypeDaily,
		WindowStart: wStart,
		WindowEnd:   wEnd,
	})
	require.NoError(t, err)
	require.Equal(t, model.ReconciliationStatusMatch, rec.Status, "msg=%s raw=%s local=%v upstream=%v",
		rec.Message, rec.UpstreamRawJSON, rec.LocalUsedUSD, rec.UpstreamUsedUSD)
	require.InDelta(t, 1.0, rec.LocalUsedUSD, 1e-9)
}

func TestRunOnce_Sub2Api_BalanceSnapshot_Mismatch(t *testing.T) {
	resetReconcilTables(t)
	srv := startFakeSub2Api()
	defer srv.Close()

	// 本地累计 500_000 quota = $1，上游 quota.used = $5 → delta = $4 → mismatch
	seedChannel(t, 202, "sk-mismatch-1", srv.URL, 500_000)
	seedConfig(t, 202, model.ReconciliationUpstreamTypeSub2Api)

	rec, err := RunOnce(context.Background(), RunOptions{ChannelId: 202, RunType: model.ReconciliationRunTypeBalance})
	require.NoError(t, err)
	require.Equal(t, model.ReconciliationStatusMismatch, rec.Status)
	require.InDelta(t, 4.0, rec.DeltaUSD, 1e-9)
}

func TestSumChannelConsumeQuota(t *testing.T) {
	resetReconcilTables(t)
	now := time.Now()
	// 三条 log: 两条在窗口内 (Consume)，一条窗口外
	logs := []model.Log{
		{Type: model.LogTypeConsume, ChannelId: 301, Quota: 100_000, CreatedAt: now.Add(-30 * time.Minute).Unix()},
		{Type: model.LogTypeConsume, ChannelId: 301, Quota: 200_000, CreatedAt: now.Add(-10 * time.Minute).Unix()},
		{Type: model.LogTypeConsume, ChannelId: 301, Quota: 999_999, CreatedAt: now.Add(-2 * time.Hour).Unix()},
		// 不同 channel 不应被计入
		{Type: model.LogTypeConsume, ChannelId: 302, Quota: 999_999, CreatedAt: now.Add(-30 * time.Minute).Unix()},
		// 非 Consume 类型不应被计入
		{Type: model.LogTypeTopup, ChannelId: 301, Quota: 999_999, CreatedAt: now.Add(-30 * time.Minute).Unix()},
	}
	for i := range logs {
		require.NoError(t, model.DB.Create(&logs[i]).Error)
	}

	sum, err := model.SumChannelConsumeQuota(301, now.Add(-1*time.Hour).Unix(), now.Unix()+1)
	require.NoError(t, err)
	require.Equal(t, int64(300_000), sum)
}

func TestSafeDeltaRel(t *testing.T) {
	require.InDelta(t, 0.5, SafeDeltaRel(1.0, 2.0), 1e-9)
	require.InDelta(t, 1.0, SafeDeltaRel(2.0, 0.0), 1e-9)
	require.InDelta(t, 0.0, SafeDeltaRel(0.0, 0.0), 1e-9)
}
