package controller

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func GetAvailabilityOverview(c *gin.Context) {
	data, err := service.GetGroupAvailabilityOverview()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetAvailabilityGroup(c *gin.Context) {
	group := strings.TrimSpace(c.Query("group"))
	if group == "" {
		common.ApiErrorMsg(c, "group is required")
		return
	}
	rangeKey := strings.TrimSpace(c.Query("range"))
	data, err := service.GetGroupAvailabilityTimeseries(group, rangeKey)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}
