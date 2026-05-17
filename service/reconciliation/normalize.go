package reconciliation

import (
	"github.com/QuantumNous/new-api/common"
)

// QuotaToUSD 把 new-api 内部的 quota 整数（500_000 quota = $1）换算为 USD。
// 复用 common.QuotaPerUnit 全局变量；该变量在站点启动后保持不变。
func QuotaToUSD(quota float64) float64 {
	if common.QuotaPerUnit <= 0 {
		return 0
	}
	return quota / common.QuotaPerUnit
}

// USDToQuota 把 USD 金额反向换算为 quota 整数，向下取整由调用方决定。
func USDToQuota(usd float64) float64 {
	return usd * common.QuotaPerUnit
}

// SafeDeltaRel 计算相对偏差 |delta| / |reference|，对 reference≈0 做防呆。
// 当 reference 为 0 而 delta 非 0 时返回一个足够大的相对值（1.0 表示 100%），
// 避免阈值判断在零基准下出现 0/0 → NaN 的歧义。
func SafeDeltaRel(delta, reference float64) float64 {
	abs := delta
	if abs < 0 {
		abs = -abs
	}
	refAbs := reference
	if refAbs < 0 {
		refAbs = -refAbs
	}
	if refAbs < 1e-9 {
		if abs < 1e-9 {
			return 0
		}
		return 1.0
	}
	return abs / refAbs
}
