package audit

import (
	"net/http"
	"strconv"

	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/gin-gonic/gin"
)

// ListOperationLogs returns a page of console operation audit entries.
func ListOperationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	logs, total, err := model.ListOperationLogs(
		page,
		size,
		c.Query("username"),
		c.Query("module"),
		c.Query("cluster"),
		c.Query("method"),
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": logs, "total": total})
}
