package controller

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/gin-gonic/gin"
)

// PublicStatusPage 公开服务状态页 - 无需认证
func PublicStatusPage(c *gin.Context) {
	channels, _ := model.GetAllChannels(0, 1000, true, true)

	type ChannelStatus struct {
		ID           int      `json:"id"`
		Name         string   `json:"name"`
		Provider     int      `json:"provider"`
		Status       string   `json:"status"`
		ResponseTime int      `json:"response_time_ms"`
		Models       []string `json:"models"`
	}

	var statuses []ChannelStatus
	totalChannels := len(channels)
	healthyCount := 0
	degradedCount := 0
	downCount := 0

	for _, ch := range channels {
		var modelList []string
		if ch.Models != "" {
			for _, m := range strings.Split(ch.Models, ",") {
				modelList = append(modelList, strings.TrimSpace(m))
			}
		}

		if ch.Status != common.ChannelStatusEnabled {
			downCount++
			statuses = append(statuses, ChannelStatus{
				ID:       ch.Id,
				Name:     ch.Name,
				Provider: ch.Type,
				Status:   "down",
				Models:   modelList,
			})
			continue
		}

		var status string
		if ch.ResponseTime == 0 {
			status = "unknown"
		} else if ch.ResponseTime < 2000 {
			status = "operational"
			healthyCount++
		} else if ch.ResponseTime < 5000 {
			status = "degraded"
			degradedCount++
		} else {
			status = "down"
			downCount++
		}

		statuses = append(statuses, ChannelStatus{
			ID:           ch.Id,
			Name:         ch.Name,
			Provider:     ch.Type,
			Status:       status,
			ResponseTime: ch.ResponseTime,
			Models:       modelList,
		})
	}

	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].ID < statuses[j].ID
	})

	overallStatus := "operational"
	if downCount > 0 {
		overallStatus = "major_outage"
	} else if degradedCount > 0 {
		overallStatus = "partial_outage"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"overall_status": overallStatus,
			"updated_at":     time.Now().Unix(),
			"total_channels": totalChannels,
			"healthy":        healthyCount,
			"degraded":       degradedCount,
			"down":           downCount,
			"channels":       statuses,
		},
	})
}
