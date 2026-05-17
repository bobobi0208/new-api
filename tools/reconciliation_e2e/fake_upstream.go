// Package main 是"上游对账"功能的伪上游 + 端到端验证工具，仅供本地手动跑通后端全链路使用。
//
// 启动方式：
//
//	go run ./tools/reconciliation_e2e/fake_upstream.go
//
// 监听 :18001 端口，同时实现两类路径：
//
//	/api/usage/token/              new-api 上游格式（quota 整数）
//	/dashboard/billing/subscription new-api 兜底端点
//	/v1/usage                       sub2api 上游格式（USD）
//
// 通过请求头里的 Authorization: Bearer 区分 token，按 token 后缀决定响应场景：
//
//	sk-match-*       上游 used = 本地预期 used（应产生 match）
//	sk-mismatch-*    上游 used 故意比本地高一截（应产生 mismatch）
//	sk-inconc-*      上游故意返回不可解析字段（应产生 inconclusive）
//	sk-err-*         上游返回 500
//	其他              默认 match 场景
//
// 本程序只是 helper，不进编译主二进制。
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/usage/token/", handleNewApiUsage)
	mux.HandleFunc("/dashboard/billing/subscription", handleNewApiSubscription)
	mux.HandleFunc("/v1/usage", handleSub2ApiUsage)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("ok")) })

	addr := ":18001"
	log.Printf("fake upstream listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func extractScenario(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	switch {
	case strings.HasPrefix(token, "sk-mismatch"):
		return "mismatch"
	case strings.HasPrefix(token, "sk-inconc"):
		return "inconclusive"
	case strings.HasPrefix(token, "sk-err"):
		return "error"
	}
	return "match"
}

// new-api 上游：返回 quota 整数（500_000 = $1）
// 这里假设本地累计 used_quota = 500_000（= $1）；
//
//	match    场景：上游 total_used = 500_000   → delta = 0
//	mismatch 场景：上游 total_used = 1_500_000 → delta = +$2 USD
//	inconc   场景：缺 data 字段
func handleNewApiUsage(w http.ResponseWriter, r *http.Request) {
	scenario := extractScenario(r)
	if scenario == "error" {
		http.Error(w, "boom", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if scenario == "inconclusive" {
		_ = json.NewEncoder(w).Encode(map[string]any{"code": true, "message": "ok"})
		return
	}
	used := 500_000.0
	available := 9_500_000.0
	if scenario == "mismatch" {
		used = 1_500_000.0
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    true,
		"message": "ok",
		"data": map[string]any{
			"object":          "token_usage",
			"name":            "fake-token",
			"total_granted":   used + available,
			"total_used":      used,
			"total_available": available,
			"unlimited_quota": false,
			"expires_at":      0,
		},
	})
}

// new-api 上游兜底 /dashboard/billing/subscription
// 返回 hard_limit_usd / soft_limit_usd（远端 OpenAI 风格）
func handleNewApiSubscription(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"object":                "billing_subscription",
		"has_payment_method":    true,
		"soft_limit_usd":        19.0,
		"hard_limit_usd":        20.0,
		"system_hard_limit_usd": 20.0,
		"access_until":          0,
	})
}

// sub2api 上游：返回 USD
// 本地累计 used_quota = 500_000（= $1）作为对账基准：
//
//	match    场景：quota.used = 1.0  → delta = 0
//	mismatch 场景：quota.used = 5.0  → delta = +$4
//	inconc   场景：返回 mode 但所有金额字段缺失
//
// 窗口请求（带 start_date/end_date）下，usage.today.cost 用作窗口聚合金额。
func handleSub2ApiUsage(w http.ResponseWriter, r *http.Request) {
	scenario := extractScenario(r)
	if scenario == "error" {
		http.Error(w, "boom", http.StatusInternalServerError)
		return
	}
	hasWindow := r.URL.Query().Get("start_date") != ""

	resp := map[string]any{
		"mode":    "quota_limited",
		"isValid": true,
		"status":  "active",
	}
	if scenario == "inconclusive" {
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	used := 1.0
	if scenario == "mismatch" {
		used = 5.0
	}
	resp["quota"] = map[string]any{
		"limit":     20.0,
		"used":      used,
		"remaining": 20.0 - used,
		"unit":      "USD",
	}
	resp["remaining"] = 20.0 - used
	resp["unit"] = "USD"

	todayCost := used
	if scenario == "mismatch" && hasWindow {
		todayCost = 4.0
	}
	resp["usage"] = map[string]any{
		"today": map[string]any{
			"requests":     10,
			"input_tokens": 1000,
			"output_tokens": 500,
			"total_tokens": 1500,
			"cost":         todayCost,
			"actual_cost":  todayCost,
		},
		"total": map[string]any{
			"requests":     100,
			"input_tokens": 10000,
			"output_tokens": 5000,
			"total_tokens": 15000,
			"cost":         used,
			"actual_cost":  used,
		},
	}

	if scenario == "match" && hasWindow {
		fmt.Fprintf(w, "")
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
