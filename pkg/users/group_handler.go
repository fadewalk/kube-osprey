package users

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/fadewalk/kube-osprey/pkg/common"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/fadewalk/kube-osprey/pkg/rbac"
	"github.com/gin-gonic/gin"
)

type createGroupReq struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
}

type updateGroupReq struct {
	Description *string `json:"description"`
	Enabled     *bool   `json:"enabled"`
}

type addMembersReq struct {
	UserIDs   []uint     `json:"userIds" binding:"required"`
	ExpiresAt *time.Time `json:"expiresAt"`
}

// ListUserGroups returns all groups with members preloaded.
func ListUserGroups(c *gin.Context) {
	var groups []model.UserGroup
	if err := model.DB.Preload("Members.User").Find(&groups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list groups"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"groups": groups})
}

// CreateUserGroup creates a locally managed group.
func CreateUserGroup(c *gin.Context) {
	if common.IsSectionManaged("rbac") {
		c.JSON(http.StatusForbidden, gin.H{"error": "rbac is managed by configuration"})
		return
	}
	var req createGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	name := strings.TrimSpace(req.Name)
	if len(name) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group name must be at least 2 characters"})
		return
	}
	if _, err := model.GetUserGroupByName(name); err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "group already exists"})
		return
	}
	group := model.UserGroup{Name: name, Description: req.Description, Enabled: true}
	if err := model.DB.Create(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create group"})
		return
	}
	// Group names become role-assignment subjects; keep the RBAC cache fresh.
	rbac.TriggerSync()
	c.JSON(http.StatusCreated, group)
}

// UpdateUserGroup toggles enablement or edits the description. The name is
// immutable because role assignments reference it.
func UpdateUserGroup(c *gin.Context) {
	if common.IsSectionManaged("rbac") {
		c.JSON(http.StatusForbidden, gin.H{"error": "rbac is managed by configuration"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	var group model.UserGroup
	if err := model.DB.First(&group, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	var req updateGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Description != nil {
		group.Description = *req.Description
	}
	if req.Enabled != nil {
		group.Enabled = *req.Enabled
	}
	if err := model.DB.Save(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update group"})
		return
	}
	rbac.TriggerSync()
	c.JSON(http.StatusOK, group)
}

// DeleteUserGroup removes the group; its role assignments (subjectType=group,
// subject=name) keep dangling until roles are reassigned, so remove them too.
func DeleteUserGroup(c *gin.Context) {
	if common.IsSectionManaged("rbac") {
		c.JSON(http.StatusForbidden, gin.H{"error": "rbac is managed by configuration"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	var group model.UserGroup
	if err := model.DB.First(&group, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	if err := model.DB.Delete(&group).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete group"})
		return
	}
	if err := model.DB.Where("subject_type = ? AND subject = ?", model.SubjectTypeGroup, group.Name).
		Delete(&model.RoleAssignment{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete group role assignments"})
		return
	}
	rbac.TriggerSync()
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// AddGroupMembers binds users to the group, optionally with an expiry date.
// Users already in the group keep their existing binding.
func AddGroupMembers(c *gin.Context) {
	if common.IsSectionManaged("rbac") {
		c.JSON(http.StatusForbidden, gin.H{"error": "rbac is managed by configuration"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	var group model.UserGroup
	if err := model.DB.First(&group, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "group not found"})
		return
	}
	var req addMembersReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var existing []model.UserGroupMember
	if err := model.DB.Where("group_id = ? AND user_id IN ?", group.ID, req.UserIDs).Find(&existing).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing members"})
		return
	}
	existingIDs := make(map[uint]bool, len(existing))
	for _, m := range existing {
		existingIDs[m.UserID] = true
	}
	created := 0
	for _, uid := range req.UserIDs {
		if existingIDs[uid] {
			continue
		}
		member := model.UserGroupMember{GroupID: group.ID, UserID: uid, ExpiresAt: req.ExpiresAt}
		if err := model.DB.Create(&member).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add members"})
			return
		}
		created++
	}
	rbac.TriggerSync()
	c.JSON(http.StatusOK, gin.H{"added": created})
}

// RemoveGroupMember deletes a single membership row.
func RemoveGroupMember(c *gin.Context) {
	if common.IsSectionManaged("rbac") {
		c.JSON(http.StatusForbidden, gin.H{"error": "rbac is managed by configuration"})
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
		return
	}
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member id"})
		return
	}
	if err := model.DB.Where("group_id = ? AND id = ?", uint(id), uint(memberID)).
		Delete(&model.UserGroupMember{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to remove member"})
		return
	}
	rbac.TriggerSync()
	c.JSON(http.StatusOK, gin.H{"success": true})
}
