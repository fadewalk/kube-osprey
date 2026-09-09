package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	prommodel "github.com/prometheus/common/model"
	"github.com/zxh326/kite/pkg/cluster"
	"github.com/zxh326/kite/pkg/model"
	"github.com/zxh326/kite/pkg/rbac"
)

const (
	maxPromQLLength = 8192
	maxPromQLRange  = 24 * time.Hour
	maxPromQLStep   = 15 * time.Second
	maxPromQLPoints = 11000
)

type promQLResponse struct {
	ResultType string          `json:"resultType"`
	Result     json.RawMessage `json:"result"`
	Warnings   []string        `json:"warnings,omitempty"`
}

func (h *Handler) QueryPromQL(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)
	if !rbac.CanAccessCluster(user, cs.Name) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	if cs.PromClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Prometheus client not available"})
		return
	}
	query := strings.TrimSpace(c.Query("query"))
	if err := validatePromQL(query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	ts, err := parsePromTime(c.Query("time"), time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid time"})
		return
	}
	ctx, cancel := contextWithPromQLTimeout(c)
	defer cancel()
	value, warnings, err := cs.PromClient.Query(ctx, query, ts)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "warnings": warnings})
		return
	}
	writePromQLResponse(c, value, warnings)
}

func (h *Handler) QueryPromQLRange(c *gin.Context) {
	cs := c.MustGet("cluster").(*cluster.ClientSet)
	user := c.MustGet("user").(model.User)
	if !rbac.CanAccessCluster(user, cs.Name) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}
	if cs.PromClient == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Prometheus client not available"})
		return
	}
	query := strings.TrimSpace(c.Query("query"))
	if err := validatePromQL(query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	end, err := parsePromTime(c.Query("end"), time.Now())
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end time"})
		return
	}
	start, err := parsePromTime(c.Query("start"), end.Add(-time.Hour))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start time"})
		return
	}
	step, err := parsePromStep(c.Query("step"))
	if err != nil || !start.Before(end) || end.Sub(start) > maxPromQLRange || int(end.Sub(start)/step)+1 > maxPromQLPoints {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid range or step; range must be at most 24h and contain at most 11000 points"})
		return
	}
	ctx, cancel := contextWithPromQLTimeout(c)
	defer cancel()
	value, warnings, err := cs.PromClient.QueryRange(ctx, query, promv1.Range{Start: start, End: end, Step: step})
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "warnings": warnings})
		return
	}
	writePromQLResponse(c, value, warnings)
}

func validatePromQL(query string) error {
	if query == "" {
		return fmt.Errorf("query is required")
	}
	if len(query) > maxPromQLLength {
		return fmt.Errorf("query exceeds the %d character limit", maxPromQLLength)
	}
	return nil
}

func parsePromTime(value string, fallback time.Time) (time.Time, error) {
	if value == "" {
		return fallback, nil
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil {
		return time.Unix(0, int64(seconds*float64(time.Second))), nil
	}
	return time.Parse(time.RFC3339, value)
}

func parsePromStep(value string) (time.Duration, error) {
	if value == "" {
		return time.Minute, nil
	}
	if seconds, err := strconv.ParseFloat(value, 64); err == nil {
		step := time.Duration(seconds * float64(time.Second))
		if step >= maxPromQLStep {
			return step, nil
		}
	}
	return 0, fmt.Errorf("step must be at least 15 seconds")
}

func contextWithPromQLTimeout(c *gin.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), 30*time.Second)
}

func writePromQLResponse(c *gin.Context, value prommodel.Value, warnings []string) {
	result, err := json.Marshal(value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encode Prometheus result"})
		return
	}
	c.JSON(http.StatusOK, promQLResponse{ResultType: value.Type().String(), Result: result, Warnings: warnings})
}
