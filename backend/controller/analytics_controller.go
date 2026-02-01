package controller

import (
	"net/http"
	"time"

	"backend/entity"
	"backend/repository"

	"github.com/gin-gonic/gin"
)

// AnalyticsController 管理员分析控制器（需 can_view_analytics 权限）
type AnalyticsController struct {
	analyticsRepo repository.AnalyticsRepository
	contentRepo   repository.ContentRepository
	purchaseRepo  repository.PurchaseRepository
}

// NewAnalyticsController 创建分析控制器
func NewAnalyticsController(
	analyticsRepo repository.AnalyticsRepository,
	contentRepo repository.ContentRepository,
	purchaseRepo repository.PurchaseRepository,
) *AnalyticsController {
	return &AnalyticsController{
		analyticsRepo: analyticsRepo,
		contentRepo:   contentRepo,
		purchaseRepo:  purchaseRepo,
	}
}

// GetAdminVideoAnalytics 管理员视频分析
// GET /api/admin/analytics/video?start_date=&end_date=
func (ac *AnalyticsController) GetAdminVideoAnalytics(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")
	if startDate == "" || endDate == "" {
		endDate = time.Now().Format("2006-01-02")
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}

	totalViews, totalPlayCount, totalWatchTime, err := ac.analyticsRepo.GetAdminVideoStats(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get video stats",
			"details": err.Error(),
		})
		return
	}

	topContents, err := ac.analyticsRepo.GetAdminTopVideoContents(15, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get top contents",
			"details": err.Error(),
		})
		return
	}

	// 附上内容名称
	topWithNames := make([]map[string]interface{}, 0, len(topContents))
	for _, stat := range topContents {
		item := map[string]interface{}{
			"content_id":       stat.ContentID,
			"total_views":     stat.TotalViews,
			"total_play_count": stat.TotalPlayCount,
			"total_watch_time": stat.TotalWatchTime,
			"total_revenue":   stat.TotalRevenue,
			"name":            "",
		}
		if content, err := ac.contentRepo.GetByID(stat.ContentID); err == nil {
			item["name"] = content.Name
		}
		topWithNames = append(topWithNames, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"start_date":       startDate,
		"end_date":         endDate,
		"total_views":      totalViews,
		"total_play_count": totalPlayCount,
		"total_watch_time": totalWatchTime,
		"top_contents":    topWithNames,
	})
}

// GetAdminRevenueAnalytics 管理员收入分析
// GET /api/admin/analytics/revenue?start_date=&end_date=
func (ac *AnalyticsController) GetAdminRevenueAnalytics(c *gin.Context) {
	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")
	if startDate == "" || endDate == "" {
		endDate = time.Now().Format("2006-01-02")
		startDate = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}

	// 内容收入（analytics 表中按内容售出积分对应的收入）
	contentRevenue, err := ac.analyticsRepo.GetAdminRevenueFromAnalytics(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get content revenue",
			"details": err.Error(),
		})
		return
	}

	// 支付网关收入：用户购买积分（credits）的金额
	creditsRevenue, err := ac.purchaseRepo.GetTotalRevenueByDateRange(entity.PurchaseTypeCredits, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get credits revenue",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"start_date":      startDate,
		"end_date":        endDate,
		"content_revenue": contentRevenue,
		"credits_revenue": creditsRevenue,
		"total_revenue":   contentRevenue + creditsRevenue,
	})
}
