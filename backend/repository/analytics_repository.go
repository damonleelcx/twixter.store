package repository

import (
	"backend/entity"
	"time"

	"gorm.io/gorm"
)

// AnalyticsRepository 分析数据仓库接口
type AnalyticsRepository interface {
	// 基础 CRUD 操作
	Create(analytics *entity.Analytics) error
	GetByID(id uint) (*entity.Analytics, error)
	GetByContentAndDate(contentID uint, date string) (*entity.Analytics, error)
	Update(analytics *entity.Analytics) error
	Delete(id uint) error

	// 查询操作
	GetByContentID(contentID uint, limit, offset int) ([]entity.Analytics, error)
	GetByDateRange(startDate, endDate string, limit, offset int) ([]entity.Analytics, error)
	GetByContentAndDateRange(contentID uint, startDate, endDate string) ([]entity.Analytics, error)
	GetTopContentsByViews(limit int, startDate, endDate string) ([]entity.Analytics, error)
	GetTopContentsByLikes(limit int, startDate, endDate string) ([]entity.Analytics, error)

	// 统计操作
	IncrementViews(contentID uint) error
	IncrementUniqueViews(contentID uint) error
	IncrementLikes(contentID uint) error
	IncrementDislikes(contentID uint) error
	IncrementShares(contentID uint) error
	IncrementComments(contentID uint) error
	IncrementDownloads(contentID uint) error
	IncrementPlayCount(contentID uint) error
	AddWatchTime(contentID uint, seconds float64) error
	UpdateCompletionRate(contentID uint, contentDuration float64) error
	IncrementPurchaseCount(contentID uint) error
	AddRevenue(contentID uint, amount float64) error

	// 聚合统计
	GetTotalViews(contentID uint, startDate, endDate string) (int64, error)
	GetTotalLikes(contentID uint, startDate, endDate string) (int64, error)
	GetAverageWatchTime(contentID uint, startDate, endDate string) (float64, error)
	GetTotalRevenue(contentID uint, startDate, endDate string) (float64, error)

	// Admin 全站统计
	GetAdminVideoStats(startDate, endDate string) (totalViews, totalPlayCount int64, totalWatchTime float64, err error)
	GetAdminRevenueFromAnalytics(startDate, endDate string) (float64, error)
	GetAdminTopVideoContents(limit int, startDate, endDate string) ([]AdminVideoContentStat, error)
}

// AdminVideoContentStat 管理员视频内容统计（按 content_id 聚合）
type AdminVideoContentStat struct {
	ContentID     uint    `json:"content_id"`
	TotalViews    int64   `json:"total_views"`
	TotalPlayCount int64  `json:"total_play_count"`
	TotalWatchTime float64 `json:"total_watch_time"`
	TotalRevenue  float64 `json:"total_revenue"`
}

// analyticsRepository 分析数据仓库实现
type analyticsRepository struct {
	db *gorm.DB
}

// NewAnalyticsRepository 创建分析数据仓库实例
func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

// Create 创建分析数据
func (r *analyticsRepository) Create(analytics *entity.Analytics) error {
	return r.db.Create(analytics).Error
}

// GetByID 根据ID获取分析数据
func (r *analyticsRepository) GetByID(id uint) (*entity.Analytics, error) {
	var analytics entity.Analytics
	if err := r.db.First(&analytics, id).Error; err != nil {
		return nil, err
	}
	return &analytics, nil
}

// GetByContentAndDate 根据内容ID和日期获取分析数据
func (r *analyticsRepository) GetByContentAndDate(contentID uint, date string) (*entity.Analytics, error) {
	var analytics entity.Analytics
	if err := r.db.Where("content_id = ? AND date = ?", contentID, date).
		First(&analytics).Error; err != nil {
		return nil, err
	}
	return &analytics, nil
}

// Update 更新分析数据
func (r *analyticsRepository) Update(analytics *entity.Analytics) error {
	return r.db.Save(analytics).Error
}

// Delete 删除分析数据（软删除）
func (r *analyticsRepository) Delete(id uint) error {
	return r.db.Delete(&entity.Analytics{}, id).Error
}

// GetByContentID 根据内容ID获取分析数据列表
func (r *analyticsRepository) GetByContentID(contentID uint, limit, offset int) ([]entity.Analytics, error) {
	var analytics []entity.Analytics
	query := r.db.Where("content_id = ?", contentID).
		Limit(limit).Offset(offset).Order("date DESC")
	if err := query.Find(&analytics).Error; err != nil {
		return nil, err
	}
	return analytics, nil
}

// GetByDateRange 根据日期范围获取分析数据
func (r *analyticsRepository) GetByDateRange(startDate, endDate string, limit, offset int) ([]entity.Analytics, error) {
	var analytics []entity.Analytics
	query := r.db.Where("date >= ? AND date <= ?", startDate, endDate).
		Limit(limit).Offset(offset).Order("date DESC")
	if err := query.Find(&analytics).Error; err != nil {
		return nil, err
	}
	return analytics, nil
}

// GetByContentAndDateRange 根据内容ID和日期范围获取分析数据
func (r *analyticsRepository) GetByContentAndDateRange(contentID uint, startDate, endDate string) ([]entity.Analytics, error) {
	var analytics []entity.Analytics
	query := r.db.Where("content_id = ? AND date >= ? AND date <= ?", contentID, startDate, endDate).
		Order("date ASC")
	if err := query.Find(&analytics).Error; err != nil {
		return nil, err
	}
	return analytics, nil
}

// GetTopContentsByViews 获取查看次数最多的内容
func (r *analyticsRepository) GetTopContentsByViews(limit int, startDate, endDate string) ([]entity.Analytics, error) {
	var analytics []entity.Analytics
	query := r.db.Where("date >= ? AND date <= ?", startDate, endDate).
		Order("views DESC").Limit(limit)
	if err := query.Find(&analytics).Error; err != nil {
		return nil, err
	}
	return analytics, nil
}

// GetTopContentsByLikes 获取点赞次数最多的内容
func (r *analyticsRepository) GetTopContentsByLikes(limit int, startDate, endDate string) ([]entity.Analytics, error) {
	var analytics []entity.Analytics
	query := r.db.Where("date >= ? AND date <= ?", startDate, endDate).
		Order("likes DESC").Limit(limit)
	if err := query.Find(&analytics).Error; err != nil {
		return nil, err
	}
	return analytics, nil
}

// getOrCreateAnalytics 获取或创建当天的分析数据
func (r *analyticsRepository) getOrCreateAnalytics(contentID uint) (*entity.Analytics, error) {
	today := time.Now().Format("2006-01-02")
	analytics, err := r.GetByContentAndDate(contentID, today)
	if err == nil {
		return analytics, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// 创建新的分析数据
	newAnalytics := &entity.Analytics{
		ContentID: contentID,
		Date:      today,
	}
	if err := r.Create(newAnalytics); err != nil {
		return nil, err
	}
	return newAnalytics, nil
}

// IncrementViews 增加查看次数
func (r *analyticsRepository) IncrementViews(contentID uint) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("views", gorm.Expr("views + ?", 1)).Error
}

// IncrementUniqueViews 增加独立访客数
func (r *analyticsRepository) IncrementUniqueViews(contentID uint) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("unique_views", gorm.Expr("unique_views + ?", 1)).Error
}

// IncrementLikes 增加点赞次数
func (r *analyticsRepository) IncrementLikes(contentID uint) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("likes", gorm.Expr("likes + ?", 1)).Error
}

// IncrementDislikes 增加点踩次数
func (r *analyticsRepository) IncrementDislikes(contentID uint) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("dislikes", gorm.Expr("dislikes + ?", 1)).Error
}

// IncrementShares 增加分享次数
func (r *analyticsRepository) IncrementShares(contentID uint) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("shares", gorm.Expr("shares + ?", 1)).Error
}

// IncrementComments 增加评论次数
func (r *analyticsRepository) IncrementComments(contentID uint) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("comments", gorm.Expr("comments + ?", 1)).Error
}

// IncrementDownloads 增加下载次数
func (r *analyticsRepository) IncrementDownloads(contentID uint) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("downloads", gorm.Expr("downloads + ?", 1)).Error
}

// IncrementPlayCount 增加播放次数
func (r *analyticsRepository) IncrementPlayCount(contentID uint) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("play_count", gorm.Expr("play_count + ?", 1)).Error
}

// AddWatchTime 添加观看时长
func (r *analyticsRepository) AddWatchTime(contentID uint, seconds float64) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}

	// 更新总观看时长和平均观看时长
	return r.db.Model(analytics).Updates(map[string]interface{}{
		"total_watch_time": gorm.Expr("total_watch_time + ?", seconds),
		"average_watch_time": gorm.Expr(
			"CASE WHEN play_count > 0 THEN (total_watch_time + ?) / play_count ELSE 0 END",
			seconds,
		),
	}).Error
}

// UpdateCompletionRate 根据当前平均观看时长与视频总时长更新完成率（0-1）
func (r *analyticsRepository) UpdateCompletionRate(contentID uint, contentDuration float64) error {
	if contentDuration <= 0 {
		return nil
	}
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn(
		"completion_rate",
		gorm.Expr("LEAST((total_watch_time / NULLIF(play_count, 0)) / ?, 1.0)", contentDuration),
	).Error
}

// IncrementPurchaseCount 增加购买次数
func (r *analyticsRepository) IncrementPurchaseCount(contentID uint) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("purchase_count", gorm.Expr("purchase_count + ?", 1)).Error
}

// AddRevenue 增加收入
func (r *analyticsRepository) AddRevenue(contentID uint, amount float64) error {
	analytics, err := r.getOrCreateAnalytics(contentID)
	if err != nil {
		return err
	}
	return r.db.Model(analytics).UpdateColumn("revenue", gorm.Expr("revenue + ?", amount)).Error
}

// GetTotalViews 获取总查看次数
func (r *analyticsRepository) GetTotalViews(contentID uint, startDate, endDate string) (int64, error) {
	var total int64
	query := r.db.Model(&entity.Analytics{}).
		Where("content_id = ? AND date >= ? AND date <= ?", contentID, startDate, endDate).
		Select("COALESCE(SUM(views), 0)")
	if err := query.Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetTotalLikes 获取总点赞次数
func (r *analyticsRepository) GetTotalLikes(contentID uint, startDate, endDate string) (int64, error) {
	var total int64
	query := r.db.Model(&entity.Analytics{}).
		Where("content_id = ? AND date >= ? AND date <= ?", contentID, startDate, endDate).
		Select("COALESCE(SUM(likes), 0)")
	if err := query.Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetAverageWatchTime 获取平均观看时长
func (r *analyticsRepository) GetAverageWatchTime(contentID uint, startDate, endDate string) (float64, error) {
	var avg float64
	query := r.db.Model(&entity.Analytics{}).
		Where("content_id = ? AND date >= ? AND date <= ?", contentID, startDate, endDate).
		Select("COALESCE(AVG(average_watch_time), 0)")
	if err := query.Scan(&avg).Error; err != nil {
		return 0, err
	}
	return avg, nil
}

// GetTotalRevenue 获取总收入
func (r *analyticsRepository) GetTotalRevenue(contentID uint, startDate, endDate string) (float64, error) {
	var total float64
	query := r.db.Model(&entity.Analytics{}).
		Where("content_id = ? AND date >= ? AND date <= ?", contentID, startDate, endDate).
		Select("COALESCE(SUM(revenue), 0)")
	if err := query.Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetAdminVideoStats 管理员：全站视频统计（日期范围内）
func (r *analyticsRepository) GetAdminVideoStats(startDate, endDate string) (totalViews, totalPlayCount int64, totalWatchTime float64, err error) {
	type agg struct {
		Views     int64
		PlayCount int64
		WatchTime float64
	}
	var a agg
	if err := r.db.Model(&entity.Analytics{}).
		Where("date >= ? AND date <= ?", startDate, endDate).
		Select("COALESCE(SUM(views), 0) AS views, COALESCE(SUM(play_count), 0) AS play_count, COALESCE(SUM(total_watch_time), 0) AS watch_time").
		Scan(&a).Error; err != nil {
		return 0, 0, 0, err
	}
	return a.Views, a.PlayCount, a.WatchTime, nil
}

// GetAdminRevenueFromAnalytics 管理员：从 analytics 表汇总内容收入（日期范围内）
func (r *analyticsRepository) GetAdminRevenueFromAnalytics(startDate, endDate string) (float64, error) {
	var total float64
	if err := r.db.Model(&entity.Analytics{}).
		Where("date >= ? AND date <= ?", startDate, endDate).
		Select("COALESCE(SUM(revenue), 0)").
		Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetAdminTopVideoContents 管理员：按内容聚合的播放/观看/收入 Top N
func (r *analyticsRepository) GetAdminTopVideoContents(limit int, startDate, endDate string) ([]AdminVideoContentStat, error) {
	type row struct {
		ContentID     uint
		TotalViews    int64
		TotalPlayCount int64
		TotalWatchTime float64
		TotalRevenue  float64
	}
	var rows []row
	if err := r.db.Model(&entity.Analytics{}).
		Where("date >= ? AND date <= ?", startDate, endDate).
		Select("content_id AS content_id, COALESCE(SUM(views), 0) AS total_views, COALESCE(SUM(play_count), 0) AS total_play_count, COALESCE(SUM(total_watch_time), 0) AS total_watch_time, COALESCE(SUM(revenue), 0) AS total_revenue").
		Group("content_id").
		Order("total_views DESC").
		Limit(limit).
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]AdminVideoContentStat, 0, len(rows))
	for _, rw := range rows {
		out = append(out, AdminVideoContentStat{
			ContentID:      rw.ContentID,
			TotalViews:     rw.TotalViews,
			TotalPlayCount: rw.TotalPlayCount,
			TotalWatchTime: rw.TotalWatchTime,
			TotalRevenue:   rw.TotalRevenue,
		})
	}
	return out, nil
}
