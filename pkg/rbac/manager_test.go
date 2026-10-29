package rbac

import (
	"testing"
	"time"

	"github.com/fadewalk/kube-osprey/pkg/model"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestLoadRolesFromDB(t *testing.T) {
	originalDB := model.DB
	rwlock.RLock()
	originalConfig := RBACConfig
	rwlock.RUnlock()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		model.DB = originalDB
		rwlock.Lock()
		RBACConfig = originalConfig
		rwlock.Unlock()
		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(&model.Role{}, &model.RoleAssignment{}, &model.UserGroup{}, &model.UserGroupMember{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	role := model.Role{
		Name:       "pod-reader",
		Clusters:   model.SliceString{"prod"},
		Namespaces: model.SliceString{"default"},
		Resources:  model.SliceString{"pods"},
		Verbs:      model.SliceString{"get"},
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	assignments := []model.RoleAssignment{
		{RoleID: role.ID, SubjectType: model.SubjectTypeUser, Subject: "alice"},
		{RoleID: role.ID, SubjectType: model.SubjectTypeGroup, Subject: "developers"},
	}
	if err := db.Create(&assignments).Error; err != nil {
		t.Fatalf("create assignments: %v", err)
	}

	model.DB = db
	if err := loadRolesFromDB(); err != nil {
		t.Fatalf("load roles: %v", err)
	}

	rwlock.RLock()
	config := RBACConfig
	rwlock.RUnlock()
	if config == nil || len(config.Roles) != 1 || len(config.RoleMapping) != 2 {
		t.Fatalf("loaded config = %#v, want one role and two mappings", config)
	}
	if !CanAccess(model.User{Username: "alice"}, "pods", "get", "prod", "default") {
		t.Fatal("expected user assignment to grant access")
	}
	if !CanAccess(model.User{Username: "bob", OIDCGroups: []string{"developers"}}, "pods", "get", "prod", "default") {
		t.Fatal("expected group assignment to grant access")
	}
}

func TestRoleAssignmentExpiry(t *testing.T) {
	originalDB := model.DB
	rwlock.RLock()
	originalConfig := RBACConfig
	rwlock.RUnlock()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get database: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		model.DB = originalDB
		rwlock.Lock()
		RBACConfig = originalConfig
		rwlock.Unlock()
		_ = sqlDB.Close()
	})

	if err := db.AutoMigrate(&model.Role{}, &model.RoleAssignment{}, &model.UserGroup{}, &model.UserGroupMember{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	role := model.Role{
		Name:       "pod-reader",
		Clusters:   model.SliceString{"prod"},
		Namespaces: model.SliceString{"default"},
		Resources:  model.SliceString{"pods"},
		Verbs:      model.SliceString{"get"},
	}
	if err := db.Create(&role).Error; err != nil {
		t.Fatalf("create role: %v", err)
	}
	past := time.Now().Add(-time.Hour)
	future := time.Now().Add(time.Hour)
	assignments := []model.RoleAssignment{
		{RoleID: role.ID, SubjectType: model.SubjectTypeUser, Subject: "alice", ExpiresAt: &past},
		{RoleID: role.ID, SubjectType: model.SubjectTypeUser, Subject: "bob", ExpiresAt: &future},
		{RoleID: role.ID, SubjectType: model.SubjectTypeUser, Subject: "carol"},
		{RoleID: role.ID, SubjectType: model.SubjectTypeGroup, Subject: "expired-group", ExpiresAt: &past},
		{RoleID: role.ID, SubjectType: model.SubjectTypeGroup, Subject: "active-group"},
	}
	if err := db.Create(&assignments).Error; err != nil {
		t.Fatalf("create assignments: %v", err)
	}

	model.DB = db
	if err := loadRolesFromDB(); err != nil {
		t.Fatalf("load roles: %v", err)
	}

	cases := []struct {
		name    string
		user    model.User
		granted bool
	}{
		{"expired user assignment", model.User{Username: "alice"}, false},
		{"future expiry user assignment", model.User{Username: "bob"}, true},
		{"no expiry user assignment", model.User{Username: "carol"}, true},
		{"expired group assignment", model.User{Username: "dave", OIDCGroups: []string{"expired-group"}}, false},
		{"active group assignment", model.User{Username: "eve", OIDCGroups: []string{"active-group"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CanAccess(tc.user, "pods", "get", "prod", "default")
			if got != tc.granted {
				t.Fatalf("CanAccess = %v, want %v", got, tc.granted)
			}
		})
	}
}
