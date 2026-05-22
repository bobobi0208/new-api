package service

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	// CommissionDefaultTiersOptionKey 存储全局默认阶梯配置的 Option key（JSON 字符串）
	CommissionDefaultTiersOptionKey = "CommissionDefaultTiers"

	// CommissionMaxRateBP 佣金封顶 basis points
	CommissionMaxRateBP = 2000
)

// DefaultCommissionTiers 内置兜底阶梯。当 Option 未设置时使用。
// 单位 quota；500_000 quota = $1。
var DefaultCommissionTiers = []model.CommissionTier{
	{From: 0, RateBP: 1000},               // 0 - $10k : 10%
	{From: 10000 * 500000, RateBP: 1200},  // $10k - $30k : 12%
	{From: 30000 * 500000, RateBP: 1500},  // $30k - $100k : 15%
	{From: 100000 * 500000, RateBP: 1800}, // > $100k : 18%
}

// ApplyTier 给定客户当月累计消费 quota，按累积阶梯算佣金 quota。
// tiers 必须按 From 升序排列。返回值 = 该客户能产生的佣金 quota（向下取整）。
func ApplyTier(consumeQuota int, tiers []model.CommissionTier) int {
	if consumeQuota <= 0 || len(tiers) == 0 {
		return 0
	}
	// 确保有序
	sorted := make([]model.CommissionTier, len(tiers))
	copy(sorted, tiers)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].From < sorted[j].From })

	var commission int64
	remain := int64(consumeQuota)
	for i, t := range sorted {
		var sliceQuota int64
		if i+1 < len(sorted) {
			width := int64(sorted[i+1].From - t.From)
			if width <= 0 {
				continue
			}
			if remain <= 0 {
				break
			}
			if remain >= width {
				sliceQuota = width
			} else {
				sliceQuota = remain
			}
		} else {
			// 最后一档：剩余全部
			sliceQuota = remain
		}
		rate := t.RateBP
		if rate > CommissionMaxRateBP {
			rate = CommissionMaxRateBP
		}
		if rate < 0 {
			rate = 0
		}
		commission += sliceQuota * int64(rate) / 10000
		remain -= sliceQuota
		if remain <= 0 {
			break
		}
	}
	return int(commission)
}

// GetGlobalDefaultTiers 读取全局默认阶梯（Option 表 + 内存 OptionMap 缓存）。
func GetGlobalDefaultTiers() []model.CommissionTier {
	common.OptionMapRWMutex.RLock()
	raw, ok := common.OptionMap[CommissionDefaultTiersOptionKey]
	common.OptionMapRWMutex.RUnlock()
	if !ok || strings.TrimSpace(raw) == "" {
		return DefaultCommissionTiers
	}
	var tiers []model.CommissionTier
	if err := common.UnmarshalJsonStr(raw, &tiers); err != nil || len(tiers) == 0 {
		common.SysLog("commission: failed to parse global default tiers, fallback to built-in: " + err.Error())
		return DefaultCommissionTiers
	}
	return tiers
}

// SetGlobalDefaultTiers 持久化全局默认阶梯到 Option 表。
func SetGlobalDefaultTiers(tiers []model.CommissionTier) error {
	if len(tiers) == 0 {
		return errors.New("tiers cannot be empty")
	}
	for _, t := range tiers {
		if t.From < 0 || t.RateBP < 0 || t.RateBP > CommissionMaxRateBP {
			return fmt.Errorf("invalid tier (from=%d, rate_bp=%d); rate_bp must be in [0, %d]", t.From, t.RateBP, CommissionMaxRateBP)
		}
	}
	data, err := common.Marshal(tiers)
	if err != nil {
		return err
	}
	return model.UpdateOption(CommissionDefaultTiersOptionKey, string(data))
}

// GetSalesTiers 获取某个销售应用的阶梯。优先 user.CommissionTierConfig；其次 user.CommissionRate（固定比例 → 单档阶梯）；最后全局默认。
func GetSalesTiers(salesUser *model.User) []model.CommissionTier {
	if salesUser == nil {
		return GetGlobalDefaultTiers()
	}
	if strings.TrimSpace(salesUser.CommissionTierConfig) != "" {
		var tiers []model.CommissionTier
		if err := common.UnmarshalJsonStr(salesUser.CommissionTierConfig, &tiers); err == nil && len(tiers) > 0 {
			return tiers
		}
		common.SysLog(fmt.Sprintf("commission: sales user %d has invalid tier config, fallback to defaults", salesUser.Id))
	}
	if salesUser.CommissionRate > 0 {
		rate := salesUser.CommissionRate
		if rate > CommissionMaxRateBP {
			rate = CommissionMaxRateBP
		}
		return []model.CommissionTier{{From: 0, RateBP: rate}}
	}
	return GetGlobalDefaultTiers()
}

// MonthRange 把 "2026-05" 转成 [start, end) unix 秒区间。
func MonthRange(yearMonth string) (int64, int64, error) {
	if len(yearMonth) != 7 || yearMonth[4] != '-' {
		return 0, 0, fmt.Errorf("invalid year-month: %s, expect YYYY-MM", yearMonth)
	}
	t, err := time.ParseInLocation("2006-01", yearMonth, time.Local)
	if err != nil {
		return 0, 0, err
	}
	start := t.Unix()
	end := t.AddDate(0, 1, 0).Unix()
	return start, end, nil
}

// ListSalesCustomerIds 查询某销售名下的所有客户 user_id（包括已禁用/已删的）
func ListSalesCustomerIds(salesUserId int) ([]int, error) {
	var ids []int
	err := model.DB.Model(&model.User{}).
		Where("inviter_id = ?", salesUserId).
		Pluck("id", &ids).Error
	return ids, err
}

// MonthlyConsumeByCustomer 返回某销售在指定月份内每个客户的消费 quota 总和。
// key=customer user_id, value=sum(quota)
func MonthlyConsumeByCustomer(salesUserId int, yearMonth string) (map[int]int, error) {
	customerIds, err := ListSalesCustomerIds(salesUserId)
	if err != nil {
		return nil, err
	}
	if len(customerIds) == 0 {
		return map[int]int{}, nil
	}
	start, end, err := MonthRange(yearMonth)
	if err != nil {
		return nil, err
	}
	type agg struct {
		UserId int   `gorm:"column:user_id"`
		Total  int64 `gorm:"column:total"`
	}
	var rows []agg
	err = model.LOG_DB.Model(&model.Log{}).
		Select("user_id, SUM(quota) as total").
		Where("type = ? AND user_id IN ? AND quota > 0 AND created_at >= ? AND created_at < ?",
			model.LogTypeConsume, customerIds, start, end).
		Group("user_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[int]int, len(rows))
	for _, r := range rows {
		out[r.UserId] = int(r.Total)
	}
	return out, nil
}

// MonthlyConsumeTotal 返回某销售在指定月份内所有客户的总消费 quota。
func MonthlyConsumeTotal(salesUserId int, yearMonth string) (int, error) {
	by, err := MonthlyConsumeByCustomer(salesUserId, yearMonth)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, v := range by {
		total += v
	}
	return total, nil
}

// EstimateCommissionForSales 估算某销售指定月份的应得佣金（按当前阶梯算，不写入账单）。
func EstimateCommissionForSales(salesUserId int, yearMonth string) (totalConsume int, commission int, byCustomer map[int]int, err error) {
	byCustomer, err = MonthlyConsumeByCustomer(salesUserId, yearMonth)
	if err != nil {
		return 0, 0, nil, err
	}
	user, err := model.GetUserById(salesUserId, false)
	if err != nil {
		return 0, 0, nil, err
	}
	tiers := GetSalesTiers(user)
	for _, c := range byCustomer {
		totalConsume += c
		commission += ApplyTier(c, tiers)
	}
	return totalConsume, commission, byCustomer, nil
}

// GenerateBillsForMonth 为所有销售用户生成指定月份的 draft 账单。
// 已存在的账单（同 sales_user_id + year_month）跳过，调用方决定是否清理重跑。
func GenerateBillsForMonth(yearMonth string) (created, skipped int, err error) {
	if _, _, err = MonthRange(yearMonth); err != nil {
		return 0, 0, err
	}
	var salesUsers []*model.User
	if err = model.DB.Where("role = ?", common.RoleSalesUser).Find(&salesUsers).Error; err != nil {
		return 0, 0, err
	}
	for _, su := range salesUsers {
		if _, e := model.GetCommissionBill(su.Id, yearMonth); e == nil {
			skipped++
			continue
		}
		totalConsume, commission, byCustomer, e := EstimateCommissionForSales(su.Id, yearMonth)
		if e != nil {
			common.SysLog(fmt.Sprintf("commission: failed to estimate for sales %d: %v", su.Id, e))
			continue
		}
		detail := buildBillDetail(byCustomer, GetSalesTiers(su))
		bill := &model.CommissionBill{
			SalesUserId:       su.Id,
			YearMonth:         yearMonth,
			TotalConsumeQuota: totalConsume,
			CommissionQuota:   commission,
			Status:            model.CommissionBillStatusDraft,
			Detail:            detail,
		}
		if e := model.CreateCommissionBill(bill); e != nil {
			common.SysLog(fmt.Sprintf("commission: failed to save bill for sales %d: %v", su.Id, e))
			continue
		}
		created++
	}
	return created, skipped, nil
}

// buildBillDetail 把每个客户的消费/佣金拆分序列化成 JSON 存到 Bill.Detail
func buildBillDetail(byCustomer map[int]int, tiers []model.CommissionTier) string {
	type row struct {
		CustomerId      int `json:"customer_id"`
		ConsumeQuota    int `json:"consume_quota"`
		CommissionQuota int `json:"commission_quota"`
	}
	rows := make([]row, 0, len(byCustomer))
	customerIds := make([]int, 0, len(byCustomer))
	for cid := range byCustomer {
		customerIds = append(customerIds, cid)
	}
	sort.Ints(customerIds)
	for _, cid := range customerIds {
		consume := byCustomer[cid]
		rows = append(rows, row{
			CustomerId:      cid,
			ConsumeQuota:    consume,
			CommissionQuota: ApplyTier(consume, tiers),
		})
	}
	data, err := common.Marshal(rows)
	if err != nil {
		return ""
	}
	return string(data)
}

// FormatRateBP 用于显示："1200" -> "12.00%"
func FormatRateBP(bp int) string {
	return strconv.FormatFloat(float64(bp)/100.0, 'f', 2, 64) + "%"
}
