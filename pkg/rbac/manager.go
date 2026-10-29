package rbac

import (
	"fmt"
	"sync"
	"time"

	"github.com/fadewalk/kube-osprey/pkg/common"
	"github.com/fadewalk/kube-osprey/pkg/model"
	"k8s.io/klog/v2"
)

var (
	RBACConfig *common.RolesConfig
	once       sync.Once
	rwlock     sync.RWMutex
)

func InitRBAC() {
	once.Do(func() {
		if err := model.InitDefaultRole(); err != nil {
			panic(fmt.Sprintf("failed to init default roles: %v", err))
		}
		go SyncRolesConfig()
	})
}

// loadRolesFromDB populates RBACConfig from DB rows
func loadRolesFromDB() error {
	cfg := &common.RolesConfig{
		Roles:       []common.Role{},
		RoleMapping: []common.RoleMapping{},
	}

	var roles []model.Role
	if err := model.DB.Preload("Assignments").Find(&roles).Error; err != nil {
		return err
	}

	for _, r := range roles {
		cr := common.Role{
			Name:        r.Name,
			Description: r.Description,
			Clusters:    r.Clusters,
			Namespaces:  r.Namespaces,
			Resources:   r.Resources,
			Verbs:       r.Verbs,
		}
		cfg.Roles = append(cfg.Roles, cr)

		for _, a := range r.Assignments {
			rm := common.RoleMapping{
				Name: cr.Name,
			}
			if a.SubjectType == model.SubjectTypeUser {
				rm.Users = append(rm.Users, a.Subject)
			} else {
				rm.OIDCGroups = append(rm.OIDCGroups, a.Subject)
			}
			if a.ExpiresAt != nil {
				rm.Expiry = a.ExpiresAt
			}
			cfg.RoleMapping = append(cfg.RoleMapping, rm)
		}
	}
	groups := make(map[string]*groupInfo)
	var userGroups []model.UserGroup
	if err := model.DB.Preload("Members.User").Find(&userGroups).Error; err != nil {
		return err
	}
	for _, g := range userGroups {
		info := &groupInfo{enabled: g.Enabled, members: make(map[string]*time.Time)}
		for _, m := range g.Members {
			key := m.User.Key()
			if key == "" {
				continue
			}
			info.members[key] = m.ExpiresAt
		}
		groups[g.Name] = info
	}

	rwlock.Lock()
	RBACConfig = cfg
	localGroups = groups
	rwlock.Unlock()
	return nil
}

var (
	SyncNow = make(chan struct{}, 1)
)

// TriggerSync sends a non-blocking signal to refresh RBAC config from the database.
func TriggerSync() {
	select {
	case SyncNow <- struct{}{}:
	default:
	}
}

func SyncRolesConfig() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	TriggerSync()
	for {
		select {
		case <-ticker.C:
		case <-SyncNow:
		}
		if err := loadRolesFromDB(); err != nil {
			klog.Errorf("failed to sync rbac from db: %v", err)
		}
	}
}
