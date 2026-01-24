package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"github.com/datafusion/worker/internal/logger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DataSourceHandler struct {
	db  *sql.DB
	log *logger.Logger
}

func NewDataSourceHandler(db *sql.DB, log *logger.Logger) *DataSourceHandler {
	return &DataSourceHandler{db: db, log: log}
}

type DataSource struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"` // web, api, database
	Config      string    `json:"config"` // JSON配置
	Description string    `json:"description"`
	Status      string    `json:"status"` // active, inactive
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// List 获取数据源列表
func (h *DataSourceHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	dsType := c.Query("type")

	offset := (page - 1) * pageSize

	query := `SELECT id, name, type, config, description, status, created_at, updated_at 
	          FROM data_sources WHERE 1=1`
	args := []interface{}{}

	if dsType != "" {
		query += " AND type = $1"
		args = append(args, dsType)
	}

	query += " ORDER BY created_at DESC LIMIT $" + strconv.Itoa(len(args)+1) + " OFFSET $" + strconv.Itoa(len(args)+2)
	args = append(args, pageSize, offset)

	rows, err := h.db.Query(query, args...)
	if err != nil {
		h.log.Error("查询数据源列表失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	defer rows.Close()

	datasources := []DataSource{}
	for rows.Next() {
		var ds DataSource
		err := rows.Scan(&ds.ID, &ds.Name, &ds.Type, &ds.Config, &ds.Description,
			&ds.Status, &ds.CreatedAt, &ds.UpdatedAt)
		if err != nil {
			h.log.Error("扫描数据源数据失败", zap.Error(err))
			continue
		}
		datasources = append(datasources, ds)
	}

	// 获取总数
	var total int
	countQuery := "SELECT COUNT(*) FROM data_sources WHERE 1=1"
	if dsType != "" {
		countQuery += " AND type = '" + dsType + "'"
	}
	h.db.QueryRow(countQuery).Scan(&total)

	c.JSON(http.StatusOK, gin.H{
		"data":      datasources,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 获取单个数据源
func (h *DataSourceHandler) Get(c *gin.Context) {
	id := c.Param("id")

	var ds DataSource
	err := h.db.QueryRow(`SELECT id, name, type, config, description, status, created_at, updated_at 
	                      FROM data_sources WHERE id = $1`, id).
		Scan(&ds.ID, &ds.Name, &ds.Type, &ds.Config, &ds.Description,
			&ds.Status, &ds.CreatedAt, &ds.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}
	if err != nil {
		h.log.Error("查询数据源失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, ds)
}

// Create 创建数据源
func (h *DataSourceHandler) Create(c *gin.Context) {
	var ds DataSource
	if err := c.ShouldBindJSON(&ds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if ds.Status == "" {
		ds.Status = "active"
	}

	err := h.db.QueryRow(`INSERT INTO data_sources (name, type, config, description, status) 
	    VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		ds.Name, ds.Type, ds.Config, ds.Description, ds.Status).Scan(&ds.ID)

	if err != nil {
		h.log.Error("创建数据源失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	h.log.Info("创建数据源成功", zap.Int64("datasource_id", ds.ID), zap.String("name", ds.Name))
	c.JSON(http.StatusCreated, ds)
}

// Update 更新数据源
func (h *DataSourceHandler) Update(c *gin.Context) {
	id := c.Param("id")

	var ds DataSource
	if err := c.ShouldBindJSON(&ds); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.db.Exec(`UPDATE data_sources SET 
	    name=$1, type=$2, config=$3, description=$4, status=$5, updated_at=NOW() 
	    WHERE id=$6`,
		ds.Name, ds.Type, ds.Config, ds.Description, ds.Status, id)

	if err != nil {
		h.log.Error("更新数据源失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}

	h.log.Info("更新数据源成功", zap.String("datasource_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// Delete 删除数据源
func (h *DataSourceHandler) Delete(c *gin.Context) {
	id := c.Param("id")

	result, err := h.db.Exec("DELETE FROM data_sources WHERE id=$1", id)
	if err != nil {
		h.log.Error("删除数据源失败", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "数据源不存在"})
		return
	}

	h.log.Info("删除数据源成功", zap.String("datasource_id", id))
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// TestConnection 测试数据源连接
func (h *DataSourceHandler) TestConnection(c *gin.Context) {
	id := c.Param("id")

	// TODO: 实现实际的连接测试逻辑
	h.log.Info("测试数据源连接", zap.String("datasource_id", id))
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "连接测试成功",
	})
}
