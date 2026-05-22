package model

import (
	"errors"
	"fmt"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	CommissionBillStatusDraft     = 0
	CommissionBillStatusConfirmed = 1

	WithdrawStatusPending  = 0
	WithdrawStatusApproved = 1
	WithdrawStatusRejected = 2
	WithdrawStatusPaid     = 3
)

// CommissionTier 是一个阶梯档位。From 是该档起始的累计消费 quota，RateBP 是 basis points (1000=10%)。
// 阶梯按累积方式套用：消费金额从 From 到下一档 From 之间的部分按 RateBP 计算佣金。
type CommissionTier struct {
	From   int `json:"from"`
	RateBP int `json:"rate_bp"`
}

// CommissionBill 月度结算账单：每个销售每月一条
type CommissionBill struct {
	Id                int    `json:"id"`
	SalesUserId       int    `json:"sales_user_id" gorm:"index:idx_sales_month,priority:1;not null"`
	YearMonth         string `json:"year_month" gorm:"type:varchar(7);column:period_ym;index:idx_sales_month,priority:2;not null"`
	TotalConsumeQuota int    `json:"total_consume_quota" gorm:"type:int;default:0"`
	CommissionQuota   int    `json:"commission_quota" gorm:"type:int;default:0"`
	Status            int    `json:"status" gorm:"type:int;default:0;index"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint;default:0"`
	ConfirmedAt       int64  `json:"confirmed_at" gorm:"bigint;default:0"`
	ConfirmedBy       int    `json:"confirmed_by" gorm:"type:int;default:0"`
	Detail            string `json:"detail" gorm:"type:text"` // JSON: 每个客户的消费和分得佣金
}

func (CommissionBill) TableName() string { return "commission_bills" }

// WithdrawRequest 销售的提现申请
type WithdrawRequest struct {
	Id            int    `json:"id"`
	SalesUserId   int    `json:"sales_user_id" gorm:"index;not null"`
	AmountQuota   int    `json:"amount_quota" gorm:"type:int;default:0"`
	Status        int    `json:"status" gorm:"type:int;default:0;index"`
	AppliedAt     int64  `json:"applied_at" gorm:"bigint;default:0"`
	ReviewedAt    int64  `json:"reviewed_at" gorm:"bigint;default:0"`
	ReviewedBy    int    `json:"reviewed_by" gorm:"type:int;default:0"`
	PaidAt        int64  `json:"paid_at" gorm:"bigint;default:0"`
	PaidNote      string `json:"paid_note" gorm:"type:varchar(255);default:''"`
	RejectReason  string `json:"reject_reason" gorm:"type:varchar(255);default:''"`
	ApplicantNote string `json:"applicant_note" gorm:"type:varchar(255);default:''"`
}

func (WithdrawRequest) TableName() string { return "withdraw_requests" }

// --- CommissionBill DAO ---

func CreateCommissionBill(bill *CommissionBill) error {
	if bill == nil {
		return errors.New("bill is nil")
	}
	if bill.SalesUserId == 0 || bill.YearMonth == "" {
		return errors.New("sales_user_id and year_month are required")
	}
	bill.CreatedAt = common.GetTimestamp()
	return DB.Create(bill).Error
}

func GetCommissionBillById(id int) (*CommissionBill, error) {
	var b CommissionBill
	err := DB.First(&b, id).Error
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func GetCommissionBill(salesUserId int, yearMonth string) (*CommissionBill, error) {
	var b CommissionBill
	err := DB.Where("sales_user_id = ? AND period_ym = ?", salesUserId, yearMonth).First(&b).Error
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func ListCommissionBillsForSales(salesUserId int, page, pageSize int) ([]*CommissionBill, int64, error) {
	var list []*CommissionBill
	var total int64
	tx := DB.Model(&CommissionBill{}).Where("sales_user_id = ?", salesUserId)
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := tx.Order("period_ym desc, id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func ListCommissionBillsForAdmin(yearMonth string, status int, page, pageSize int) ([]*CommissionBill, int64, error) {
	tx := DB.Model(&CommissionBill{})
	if yearMonth != "" {
		tx = tx.Where("period_ym = ?", yearMonth)
	}
	if status >= 0 {
		tx = tx.Where("status = ?", status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*CommissionBill
	err := tx.Order("period_ym desc, sales_user_id asc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

// ConfirmCommissionBillTx 确认账单：把 commission 累加到销售 user.commission_balance，事务内完成
func ConfirmCommissionBillTx(billId, adminUserId int) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var bill CommissionBill
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&bill, billId).Error; err != nil {
			return err
		}
		if bill.Status == CommissionBillStatusConfirmed {
			return fmt.Errorf("bill %d already confirmed", billId)
		}
		bill.Status = CommissionBillStatusConfirmed
		bill.ConfirmedAt = common.GetTimestamp()
		bill.ConfirmedBy = adminUserId
		if err := tx.Save(&bill).Error; err != nil {
			return err
		}
		// 把佣金累加到销售用户
		if bill.CommissionQuota > 0 {
			err := tx.Model(&User{}).Where("id = ?", bill.SalesUserId).
				UpdateColumns(map[string]interface{}{
					"commission_balance":       gorm.Expr("commission_balance + ?", bill.CommissionQuota),
					"commission_history_total": gorm.Expr("commission_history_total + ?", bill.CommissionQuota),
				}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// --- WithdrawRequest DAO ---

func CreateWithdrawRequest(req *WithdrawRequest) error {
	if req == nil {
		return errors.New("request is nil")
	}
	if req.SalesUserId == 0 || req.AmountQuota <= 0 {
		return errors.New("sales_user_id and positive amount_quota are required")
	}
	return DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&user, req.SalesUserId).Error; err != nil {
			return err
		}
		if user.CommissionBalance < req.AmountQuota {
			return errors.New("commission balance insufficient")
		}
		// 冻结：先扣余额，等审核拒绝再退回
		err := tx.Model(&User{}).Where("id = ?", user.Id).
			UpdateColumn("commission_balance", gorm.Expr("commission_balance - ?", req.AmountQuota)).Error
		if err != nil {
			return err
		}
		req.Status = WithdrawStatusPending
		req.AppliedAt = common.GetTimestamp()
		return tx.Create(req).Error
	})
}

func GetWithdrawRequestById(id int) (*WithdrawRequest, error) {
	var r WithdrawRequest
	err := DB.First(&r, id).Error
	if err != nil {
		return nil, err
	}
	return &r, nil
}

func ListWithdrawForSales(salesUserId int, page, pageSize int) ([]*WithdrawRequest, int64, error) {
	tx := DB.Model(&WithdrawRequest{}).Where("sales_user_id = ?", salesUserId)
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*WithdrawRequest
	err := tx.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func ListWithdrawForAdmin(status int, page, pageSize int) ([]*WithdrawRequest, int64, error) {
	tx := DB.Model(&WithdrawRequest{})
	if status >= 0 {
		tx = tx.Where("status = ?", status)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []*WithdrawRequest
	err := tx.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&list).Error
	return list, total, err
}

func UpdateWithdrawStatusTx(id, adminUserId, newStatus int, note, rejectReason string) error {
	return DB.Transaction(func(tx *gorm.DB) error {
		var req WithdrawRequest
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&req, id).Error; err != nil {
			return err
		}
		now := common.GetTimestamp()
		switch newStatus {
		case WithdrawStatusApproved:
			if req.Status != WithdrawStatusPending {
				return errors.New("only pending request can be approved")
			}
			req.Status = WithdrawStatusApproved
			req.ReviewedAt = now
			req.ReviewedBy = adminUserId
		case WithdrawStatusRejected:
			if req.Status != WithdrawStatusPending {
				return errors.New("only pending request can be rejected")
			}
			req.Status = WithdrawStatusRejected
			req.ReviewedAt = now
			req.ReviewedBy = adminUserId
			req.RejectReason = rejectReason
			// 退回冻结的佣金余额
			if err := tx.Model(&User{}).Where("id = ?", req.SalesUserId).
				UpdateColumn("commission_balance", gorm.Expr("commission_balance + ?", req.AmountQuota)).Error; err != nil {
				return err
			}
		case WithdrawStatusPaid:
			if req.Status != WithdrawStatusApproved {
				return errors.New("only approved request can be marked paid")
			}
			req.Status = WithdrawStatusPaid
			req.PaidAt = now
			req.PaidNote = note
		default:
			return errors.New("invalid status transition")
		}
		return tx.Save(&req).Error
	})
}
