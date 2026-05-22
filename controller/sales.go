package controller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// thisMonth 返回 "2006-01" 格式的当前月份字符串
func thisMonth() string {
	return time.Now().Format("2006-01")
}

// salesUserOrAbort 校验当前请求用户是销售且加载完整资料。被 SalesAuth 保护的路径调用。
func salesUserOrAbort(c *gin.Context) *model.User {
	id := c.GetInt("id")
	if id == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "not authenticated"})
		c.Abort()
		return nil
	}
	user, err := model.GetUserById(id, true)
	if err != nil {
		common.ApiError(c, err)
		c.Abort()
		return nil
	}
	if user.Role < common.RoleSalesUser {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "sales role required"})
		c.Abort()
		return nil
	}
	return user
}

// GetSalesDashboard GET /api/sales/dashboard
func GetSalesDashboard(c *gin.Context) {
	user := salesUserOrAbort(c)
	if user == nil {
		return
	}
	ym := c.DefaultQuery("month", thisMonth())

	customerIds, err := service.ListSalesCustomerIds(user.Id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	totalConsume, commission, _, err := service.EstimateCommissionForSales(user.Id, ym)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"month":                    ym,
			"customer_count":           len(customerIds),
			"month_consume_quota":      totalConsume,
			"month_commission_quota":   commission,
			"commission_balance":       user.CommissionBalance,
			"commission_history_total": user.CommissionHistoryTotal,
			"tiers":                    service.GetSalesTiers(user),
			"aff_code":                 user.AffCode,
			"aff_count":                user.AffCount,
		},
	})
}

// GetSalesCustomers GET /api/sales/customers?page=1&page_size=20
func GetSalesCustomers(c *gin.Context) {
	user := salesUserOrAbort(c)
	if user == nil {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	var customers []model.User
	var total int64
	tx := model.DB.Model(&model.User{}).Where("inviter_id = ?", user.Id)
	if err := tx.Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	err := tx.Select("id, username, display_name, email, status, created_at, used_quota, quota, request_count").
		Order("id desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&customers).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     customers,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// GetSalesConsumes GET /api/sales/consumes?month=2026-05
// 返回当前销售名下所有客户在指定月份的消费聚合（按客户分组）
func GetSalesConsumes(c *gin.Context) {
	user := salesUserOrAbort(c)
	if user == nil {
		return
	}
	ym := c.DefaultQuery("month", thisMonth())
	byCustomer, err := service.MonthlyConsumeByCustomer(user.Id, ym)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	tiers := service.GetSalesTiers(user)

	type row struct {
		CustomerId      int    `json:"customer_id"`
		CustomerName    string `json:"customer_name"`
		ConsumeQuota    int    `json:"consume_quota"`
		CommissionQuota int    `json:"commission_quota"`
	}
	rows := make([]row, 0, len(byCustomer))
	cids := make([]int, 0, len(byCustomer))
	for cid := range byCustomer {
		cids = append(cids, cid)
	}
	// 取客户名（一次性 IN 查询）
	nameById := map[int]string{}
	if len(cids) > 0 {
		type idName struct {
			Id          int
			Username    string
			DisplayName string
		}
		var arr []idName
		_ = model.DB.Model(&model.User{}).Select("id, username, display_name").Where("id IN ?", cids).Find(&arr).Error
		for _, a := range arr {
			name := a.DisplayName
			if name == "" {
				name = a.Username
			}
			nameById[a.Id] = name
		}
	}
	totalConsume := 0
	totalCommission := 0
	for cid, q := range byCustomer {
		cm := service.ApplyTier(q, tiers)
		rows = append(rows, row{
			CustomerId:      cid,
			CustomerName:    nameById[cid],
			ConsumeQuota:    q,
			CommissionQuota: cm,
		})
		totalConsume += q
		totalCommission += cm
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"month":            ym,
			"items":            rows,
			"total_consume":    totalConsume,
			"total_commission": totalCommission,
			"tiers":            tiers,
		},
	})
}

// GetSalesBills GET /api/sales/bills?page=1&page_size=20
func GetSalesBills(c *gin.Context) {
	user := salesUserOrAbort(c)
	if user == nil {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	bills, total, err := model.ListCommissionBillsForSales(user.Id, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     bills,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}

// SalesWithdrawRequest POST /api/sales/withdraw
type SalesWithdrawRequest struct {
	AmountQuota int    `json:"amount_quota" binding:"required"`
	Note        string `json:"applicant_note"`
}

// PostSalesWithdraw POST /api/sales/withdraw
func PostSalesWithdraw(c *gin.Context) {
	user := salesUserOrAbort(c)
	if user == nil {
		return
	}
	var req SalesWithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.AmountQuota <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "amount_quota must be positive"})
		return
	}
	wr := &model.WithdrawRequest{
		SalesUserId:   user.Id,
		AmountQuota:   req.AmountQuota,
		ApplicantNote: req.Note,
	}
	if err := model.CreateWithdrawRequest(wr); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": wr})
}

// GetSalesWithdraws GET /api/sales/withdraws?page=1&page_size=20
func GetSalesWithdraws(c *gin.Context) {
	user := salesUserOrAbort(c)
	if user == nil {
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	list, total, err := model.ListWithdrawForSales(user.Id, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"items":     list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		},
	})
}
