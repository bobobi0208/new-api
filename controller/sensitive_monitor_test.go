package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSensitiveMonitorControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	gin.SetMode(gin.TestMode)
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.SensitiveWord{}, &model.SensitiveWordHit{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func seedSensitiveMonitorHits(t *testing.T) {
	t.Helper()

	now := time.Now().Truncate(time.Second)
	require.NoError(t, model.DB.Create(&model.SensitiveWordHit{
		RuleId:        1,
		Pattern:       "target pattern",
		Action:        model.SensitiveWordActionBlock,
		UserId:        7,
		Username:      "alice",
		TokenId:       12,
		TokenName:     "prod-token",
		ModelName:     "gpt-target",
		RequestId:     "req-target",
		Path:          "/v1/chat/completions",
		PromptSnippet: "contains target keyword",
		CreatedAt:     now,
	}).Error)
	require.NoError(t, model.DB.Create(&model.SensitiveWordHit{
		RuleId:        2,
		Pattern:       "other pattern",
		Action:        model.SensitiveWordActionMonitor,
		UserId:        8,
		Username:      "bob",
		TokenId:       13,
		TokenName:     "dev-token",
		ModelName:     "gpt-other",
		RequestId:     "req-other",
		Path:          "/v1/responses",
		PromptSnippet: "clean snippet",
		CreatedAt:     now.Add(-2 * time.Hour),
	}).Error)
}

func TestGetSensitiveMonitorHitsSupportsDetailFilters(t *testing.T) {
	setupSensitiveMonitorControllerTestDB(t)
	seedSensitiveMonitorHits(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/sensitive_monitor/hits?action=1&keyword=target&username=ali&token_name=prod&model_name=target&request_id=req-target&start_time=2000-01-01T00:00:00Z&end_time=2999-01-01T00:00:00Z",
		nil,
	)

	GetSensitiveMonitorHits(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)

	var response struct {
		Success bool `json:"success"`
		Data    struct {
			Total int                      `json:"total"`
			Items []model.SensitiveWordHit `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)
	require.Equal(t, 1, response.Data.Total)
	require.Len(t, response.Data.Items, 1)
	require.Equal(t, "req-target", response.Data.Items[0].RequestId)
}

func TestExportSensitiveMonitorHitsWritesCsv(t *testing.T) {
	setupSensitiveMonitorControllerTestDB(t)
	seedSensitiveMonitorHits(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/sensitive_monitor/hits/export?keyword=target",
		nil,
	)

	ExportSensitiveMonitorHits(ctx)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Header().Get("Content-Type"), "text/csv")
	require.Contains(t, recorder.Header().Get("Content-Disposition"), "sensitive-monitor-hits-")
	require.Contains(t, recorder.Body.String(), "target pattern")
	require.Contains(t, recorder.Body.String(), "contains target keyword")
}
