package controller

import (
	"net/http"
	"strconv"

	"backend/repository"

	"github.com/gin-gonic/gin"
)

// TagsController 标签控制器
type TagsController struct {
	tagRepo repository.TagRepository
}

// NewTagsController 创建标签控制器实例
func NewTagsController(tagRepo repository.TagRepository) *TagsController {
	return &TagsController{
		tagRepo: tagRepo,
	}
}

// SearchTags 搜索标签（用于自动完成，需 can_search_tags 权限；admin 和 dark 账户类型默认拥有）
// @Summary Search tags for autocomplete
// @Description Search tags by keyword. Requires can_search_tags permission (admin and dark account types have it by default).
// @Tags tags
// @Security BearerAuth
// @Produce json
// @Param q query string false "Search keyword"
// @Param limit query int false "Max results (default 20, max 50)"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Router /api/tags/search [get]
func (tc *TagsController) SearchTags(c *gin.Context) {
	keyword := c.DefaultQuery("q", "")
	limitStr := c.DefaultQuery("limit", "20")
	limit := 20
	if n, err := strconv.Atoi(limitStr); err == nil && n > 0 {
		limit = n
		if limit > 50 {
			limit = 50
		}
	}
	tags, err := tc.tagRepo.Search(keyword, limit, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to search tags",
			"details": err.Error(),
		})
		return
	}
	result := make([]gin.H, 0, len(tags))
	for _, t := range tags {
		result = append(result, gin.H{
			"id":   t.ID,
			"name": t.Name,
			"slug": t.Slug,
		})
	}
	c.JSON(http.StatusOK, gin.H{"tags": result})
}
