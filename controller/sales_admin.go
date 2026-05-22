package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

// PromoteUserToSales POST /api/admin/sales/users/:id/promote
func PromoteUserToSales(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid user id"})
		return
	}
	user, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user.Role >= common.RoleAdminUser {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "cannot promote admin/root to sales"})
		return
	}
	if user.Role == common.RoleSalesUser {
		c.JSON(http.StatusOK, gin.H{"success": true, "message": "already sales"})
		return
	}
	if err := model.DB.Model(&model.User{}).Where("id = ?", id).Update("role", common.RoleSalesUser).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DemoteSalesToUser POST /api/admin/sales/users/:id/demote
func DemoteSalesToUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid user id"})
		return
	}
	user, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user.Role != common.RoleSalesUser {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user is not a sales"})
		return
	}
	if err := model.DB.Model(&model.User{}).Where("id = ?", id).Update("role", common.RoleCommonUser).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// BindCustomerToSalesRequest PATCH /api/admin/sales/users/:id/bind
type BindCustomerToSalesRequest struct {
	InviterId int `json:"inviter_id"`
}

// BindCustomerToSales sets inviter_id on a customer user. Pass inviter_id=0 to unbind.
func BindCustomerToSales(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid user id"})
		return
	}
	var req BindCustomerToSalesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if req.InviterId != 0 {
		sales, err := model.GetUserById(req.InviterId, false)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "inviter not found"})
			return
		}
		if sales.Role != common.RoleSalesUser {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "target inviter is not a sales user"})
			return
		}
		if sales.Id == id {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "cannot bind user to itself"})
			return
		}
	}
	if err := model.DB.Model(&model.User{}).Where("id = ?", id).Update("inviter_id", req.InviterId).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// UpdateSalesTiersRequest PATCH /api/admin/sales/users/:id/tiers
type UpdateSalesTiersRequest struct {
	CommissionRate       int                    `json:"commission_rate"`
	CommissionTierConfig []model.CommissionTier `json:"commission_tier_config"`
}

// UpdateSalesTiers sets per-sales fixed rate or custom tier config. Empty list = clear (fall back to global).
func UpdateSalesTiers(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid user id"})
		return
	}
	var req UpdateSalesTiersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	user, err := model.GetUserById(id, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if user.Role != common.RoleSalesUser {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "user is not a sales"})
		return
	}
	if req.CommissionRate < 0 || req.CommissionRate > service.CommissionMaxRateBP {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid commission_rate"})
		return
	}
	for _, t := range req.CommissionTierConfig {
		if t.From < 0 || t.RateBP < 0 || t.RateBP > service.CommissionMaxRateBP {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid tier entry"})
			return
		}
	}
	tierJSON := ""
	if len(req.CommissionTierConfig) > 0 {
		data, err := common.Marshal(req.CommissionTierConfig)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		tierJSON = string(data)
	}
	err = model.DB.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"commission_rate":        req.CommissionRate,
		"commission_tier_config": tierJSON,
	}).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetDefaultCommissionTiers GET /api/admin/sales/commission/default-tiers
func GetDefaultCommissionTiers(c *gin.Context) {
	tiers := service.GetGlobalDefaultTiers()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": tiers})
}

// SetDefaultCommissionTiersRequest PUT /api/admin/sales/commission/default-tiers
type SetDefaultCommissionTiersRequest struct {
	Tiers []model.CommissionTier `json:"tiers" binding:"required"`
}

// SetDefaultCommissionTiers PUT /api/admin/sales/commission/default-tiers
func SetDefaultCommissionTiers(c *gin.Context) {
	var req SetDefaultCommissionTiersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.SetGlobalDefaultTiers(req.Tiers); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GenerateBillsRequest POST /api/admin/sales/commission/bills/generate
type GenerateBillsRequest struct {
	Month string `json:"month" binding:"required"`
}

// GenerateMonthlyBills POST /api/admin/sales/commission/bills/generate
func GenerateMonthlyBills(c *gin.Context) {
	var req GenerateBillsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiError(c, err)
		return
	}
	created, skipped, err := service.GenerateBillsForMonth(req.Month)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"created": created, "skipped": skipped}})
}

// ListCommissionBills GET /api/admin/sales/commission/bills?month=&status=&page=&page_size=
func ListCommissionBills(c *gin.Context) {
	month := c.Query("month")
	status, _ := strconv.Atoi(c.DefaultQuery("status", "-1"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	list, total, err := model.ListCommissionBillsForAdmin(month, status, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"items": list, "total": total, "page": page, "page_size": pageSize},
	})
}

// ConfirmCommissionBill POST /api/admin/sales/commission/bills/:id/confirm
func ConfirmCommissionBill(c *gin.Context) {
	billId, err := strconv.Atoi(c.Param("id"))
	if err != nil || billId == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid bill id"})
		return
	}
	adminId := c.GetInt("id")
	if err := model.ConfirmCommissionBillTx(billId, adminId); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListAdminWithdraws GET /api/admin/sales/withdraws?status=&page=&page_size=
func ListAdminWithdraws(c *gin.Context) {
	status, _ := strconv.Atoi(c.DefaultQuery("status", "-1"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}
	list, total, err := model.ListWithdrawForAdmin(status, page, pageSize)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"items": list, "total": total, "page": page, "page_size": pageSize},
	})
}

// WithdrawActionRequest POST /api/admin/sales/withdraws/:id/{approve|reject|paid}
type WithdrawActionRequest struct {
	Note         string `json:"note"`          // paid 用：打款流水号/备注
	RejectReason string `json:"reject_reason"` // reject 用
}

// ApproveWithdraw POST /api/admin/sales/withdraws/:id/approve
func ApproveWithdraw(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}
	adminId := c.GetInt("id")
	if err := model.UpdateWithdrawStatusTx(id, adminId, model.WithdrawStatusApproved, "", ""); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// RejectWithdraw POST /api/admin/sales/withdraws/:id/reject
func RejectWithdraw(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}
	var req WithdrawActionRequest
	_ = c.ShouldBindJSON(&req)
	adminId := c.GetInt("id")
	if err := model.UpdateWithdrawStatusTx(id, adminId, model.WithdrawStatusRejected, "", req.RejectReason); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// MarkWithdrawPaid POST /api/admin/sales/withdraws/:id/paid
func MarkWithdrawPaid(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "invalid id"})
		return
	}
	var req WithdrawActionRequest
	_ = c.ShouldBindJSON(&req)
	adminId := c.GetInt("id")
	if err := model.UpdateWithdrawStatusTx(id, adminId, model.WithdrawStatusPaid, req.Note, ""); err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ListAllSales GET /api/admin/sales/users
func ListAllSales(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}
	tx := model.DB.Model(&model.User{}).Where("role = ?", common.RoleSalesUser)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	var users []model.User
	err := tx.Select("id, username, display_name, email, commission_rate, commission_tier_config, commission_balance, commission_history_total").
		Order("id desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Find(&users).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"items": users, "total": total, "page": page, "page_size": pageSize},
	})
}
