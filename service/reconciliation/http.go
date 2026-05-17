package reconciliation

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"
)

// httpGetJSON 是 reconciliation 包内的统一 GET 调用：
//   - 复用 service.GetHttpClient() 的全局连接池 / SSRF 防护
//   - 用 context.WithTimeout 控制单次请求超时（对账每个 channel 独立超时）
//   - 自动塞入 Authorization: Bearer 头
//   - 返回原始 body 字符串和解析后的 map，供 Adapter 自行进一步取字段
func httpGetJSON(ctx context.Context, fullURL, apiKey string, timeout time.Duration) (rawBody string, parsed map[string]any, status int, err error) {
	if !strings.HasPrefix(fullURL, "http://") && !strings.HasPrefix(fullURL, "https://") {
		return "", nil, 0, fmt.Errorf("invalid url: %s", fullURL)
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, fullURL, nil)
	if err != nil {
		return "", nil, 0, err
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimPrefix(apiKey, "Bearer "))
	}
	req.Header.Set("Accept", "application/json")

	client := service.GetHttpClient()
	if client == nil {
		return "", nil, 0, errors.New("http client not initialized")
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, resp.StatusCode, err
	}
	rawBody = string(body)
	parsed = make(map[string]any)
	// 遵循 CLAUDE.md Rule 1：通过 common.Unmarshal 而非 encoding/json
	if len(body) > 0 {
		if jerr := common.Unmarshal(body, &parsed); jerr != nil {
			return rawBody, nil, resp.StatusCode, fmt.Errorf("unmarshal response: %w", jerr)
		}
	}
	return rawBody, parsed, resp.StatusCode, nil
}

// joinURL 拼接 BaseURL 与子路径，自动处理尾斜杠。
func joinURL(baseURL, subPath string) string {
	baseURL = strings.TrimRight(baseURL, "/")
	if !strings.HasPrefix(subPath, "/") {
		subPath = "/" + subPath
	}
	return baseURL + subPath
}

// asFloat 从已 parsed 的 map 任意层级取一个数字字段，兼容 json.Number / float64 / int 三种来源。
func asFloat(m map[string]any, key string) (float64, bool) {
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	}
	return 0, false
}

func asBool(m map[string]any, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, ok := v.(bool)
	return ok && b
}

func asInt64(m map[string]any, key string) int64 {
	if f, ok := asFloat(m, key); ok {
		return int64(f)
	}
	return 0
}

func asMap(m map[string]any, key string) map[string]any {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	if mm, ok := v.(map[string]any); ok {
		return mm
	}
	return nil
}
