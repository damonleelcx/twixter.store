package entity

import (
	"time"

	"gorm.io/gorm"
)

// Analytics 内容分析数据模型（累积统计）
type Analytics struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	ContentID uint   `gorm:"not null;index" json:"content_id"`     // 内容ID
	Date      string `gorm:"type:date;not null;index" json:"date"` // 统计日期（YYYY-MM-DD格式）

	// 基础统计
	Views       int64 `gorm:"default:0" json:"views"`        // 查看次数
	UniqueViews int64 `gorm:"default:0" json:"unique_views"` // 独立访客数
	Likes       int64 `gorm:"default:0" json:"likes"`        // 点赞次数
	Dislikes    int64 `gorm:"default:0" json:"dislikes"`     // 点踩次数
	Shares      int64 `gorm:"default:0" json:"shares"`       // 分享次数
	Comments    int64 `gorm:"default:0" json:"comments"`     // 评论次数
	Downloads   int64 `gorm:"default:0" json:"downloads"`    // 下载次数

	// 视频特定统计
	PlayCount        int64   `gorm:"default:0" json:"play_count"`         // 播放次数
	AverageWatchTime float64 `gorm:"default:0" json:"average_watch_time"` // 平均观看时长（秒）
	TotalWatchTime   float64 `gorm:"default:0" json:"total_watch_time"`   // 总观看时长（秒）
	CompletionRate   float64 `gorm:"default:0" json:"completion_rate"`    // 完成率（0-1）

	// 互动统计
	ClickThroughRate float64 `gorm:"default:0" json:"click_through_rate"` // 点击率（0-1）
	EngagementRate   float64 `gorm:"default:0" json:"engagement_rate"`    // 互动率（0-1）

	// 收入相关（如果内容需要付费）
	PurchaseCount int64   `gorm:"default:0" json:"purchase_count"`             // 购买次数
	Revenue       float64 `gorm:"type:decimal(10,2);default:0" json:"revenue"` // 收入（如果适用）

	// 时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"` // 软删除
}

// TableName 指定表名
func (Analytics) TableName() string {
	return "analytics"
}

// BeforeCreate 创建前的钩子函数
func (a *Analytics) BeforeCreate(tx *gorm.DB) error {
	// 如果日期未设置，使用当前日期
	if a.Date == "" {
		a.Date = time.Now().Format("2006-01-02")
	}
	return nil
}

// IncrementViews 增加查看次数
func (a *Analytics) IncrementViews() {
	a.Views++
}

// IncrementUniqueViews 增加独立访客数
func (a *Analytics) IncrementUniqueViews() {
	a.UniqueViews++
}

// IncrementLikes 增加点赞次数
func (a *Analytics) IncrementLikes() {
	a.Likes++
}

// IncrementDislikes 增加点踩次数
func (a *Analytics) IncrementDislikes() {
	a.Dislikes++
}

// IncrementShares 增加分享次数
func (a *Analytics) IncrementShares() {
	a.Shares++
}

// IncrementComments 增加评论次数
func (a *Analytics) IncrementComments() {
	a.Comments++
}

// IncrementDownloads 增加下载次数
func (a *Analytics) IncrementDownloads() {
	a.Downloads++
}

// IncrementPlayCount 增加播放次数（视频）
func (a *Analytics) IncrementPlayCount() {
	a.PlayCount++
}

// AddWatchTime 添加观看时长（视频）
func (a *Analytics) AddWatchTime(seconds float64) {
	a.TotalWatchTime += seconds
	if a.PlayCount > 0 {
		a.AverageWatchTime = a.TotalWatchTime / float64(a.PlayCount)
	}
}

// UpdateCompletionRate 更新完成率（视频）
func (a *Analytics) UpdateCompletionRate(contentDuration float64) {
	if contentDuration > 0 && a.PlayCount > 0 {
		a.CompletionRate = (a.AverageWatchTime / contentDuration)
		if a.CompletionRate > 1.0 {
			a.CompletionRate = 1.0
		}
	}
}

// IncrementPurchaseCount 增加购买次数
func (a *Analytics) IncrementPurchaseCount() {
	a.PurchaseCount++
}

// AddRevenue 增加收入
func (a *Analytics) AddRevenue(amount float64) {
	a.Revenue += amount
}

// CalculateEngagementRate 计算互动率
func (a *Analytics) CalculateEngagementRate() {
	if a.Views > 0 {
		totalEngagements := float64(a.Likes + a.Comments + a.Shares)
		a.EngagementRate = totalEngagements / float64(a.Views)
	}
}
