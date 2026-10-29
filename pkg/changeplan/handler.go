package changeplan

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/fadewalk/kube-osprey/pkg/common"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/fadewalk/kube-osprey/pkg/rbac"
)

// CanApproveChangePlans reports whether the user holds the change_plans
// approve permission evaluated cluster-wide.
func CanApproveChangePlans(user model.User) bool {
	return rbac.CanAccess(user, "change_plans", "approve", "", common.AllNamespaces)
}

func userFromGin(c *gin.Context) (model.User, bool) {
	raw, ok := c.Get("user")
	if !ok {
		return model.User{}, false
	}
	user, ok := raw.(model.User)
	return user, ok
}

// ListChangePlans serves the approval center; approvers see every plan,
// creators only their own.
func (s *Service) ListChangePlans(c *gin.Context) {
	user, ok := userFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	plans, err := s.ListPlansForUser(user, CanApproveChangePlans(user), strings.TrimSpace(c.Query("status")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"plans": plans, "canApprove": CanApproveChangePlans(user)})
}

func (s *Service) GetChangePlan(c *gin.Context) {
	user, ok := userFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	planID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
		return
	}
	plan, err := s.GetPlanForUser(uint(planID), user, CanApproveChangePlans(user))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "plan not found"})
		return
	}
	c.JSON(http.StatusOK, plan)
}

// reviewChangePlan applies the human gate shared by approve and reject: a
// real approver, one distinct from the creator, and the approve permission.
func (s *Service) reviewChangePlan(c *gin.Context, review func(approver model.User, planID uint) error, result string) {
	user, ok := userFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	if !CanApproveChangePlans(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": rbac.NoAccess(user.Key(), "approve", "change_plans", common.AllNamespaces, "")})
		return
	}
	planID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
		return
	}
	if err := review(user, uint(planID)); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": result})
}

func (s *Service) ApproveChangePlan(c *gin.Context) {
	s.reviewChangePlan(c, s.ApprovePlan, "approved")
}

func (s *Service) RejectChangePlan(c *gin.Context) {
	s.reviewChangePlan(c, s.RejectPlan, "rejected")
}

// PendingChangePlanCount powers the navigation badge.
func (s *Service) PendingChangePlanCount(c *gin.Context) {
	var count int64
	if err := model.DB.Model(&model.ChangePlan{}).
		Where("status = ?", model.ChangePlanPending).Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"count": count})
}

// RollbackChangePlan reverts an APPLIED plan from its before/after
// snapshots; approver permission required.
func (s *Service) RollbackChangePlan(c *gin.Context) {
	user, ok := userFromGin(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not authenticated"})
		return
	}
	if !CanApproveChangePlans(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": rbac.NoAccess(user.Key(), "approve", "change_plans", common.AllNamespaces, "")})
		return
	}
	planID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid plan id"})
		return
	}
	plan, err := s.RollbackPlan(c.Request.Context(), uint(planID))
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, plan)
}
