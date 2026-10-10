package syncengine

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/zxh326/kite/pkg/model"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupSyncDB(t *testing.T) {
	t.Helper()
	originalDB := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("opening test database: %v", err)
	}
	if err := db.AutoMigrate(&model.SyncObject{}, &model.SyncTask{}); err != nil {
		t.Fatalf("migrating test database: %v", err)
	}
	model.DB = db
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		model.DB = originalDB
	})
}

func row(cluster, uid, name, ns, rv string) *model.SyncObject {
	return &model.SyncObject{
		ClusterName:       cluster,
		UID:               uid,
		Kind:              "Pod",
		Resource:          "pods",
		Namespace:         ns,
		Name:              name,
		SearchName:        name,
		ResourceVersion:   rv,
		CreationTimestamp: time.Now(),
		ObjectJSON:        `{"kind":"Pod","metadata":{"name":"` + name + `","uid":"` + uid + `"}}`,
	}
}

func TestIsRVNewer(t *testing.T) {
	tests := []struct {
		candidate, current string
		want               bool
	}{
		{"", "", false},
		{"123", "", true},
		{"124", "123", true},
		{"123", "124", false},
		{"123", "123", false},
		{"abc", "abd", false},
		{"1000", "999", true}, // numeric compare beats string compare
	}
	for _, tc := range tests {
		if got := isRVNewer(tc.candidate, tc.current); got != tc.want {
			t.Errorf("isRVNewer(%q, %q) = %v, want %v", tc.candidate, tc.current, got, tc.want)
		}
	}
}

func TestUpsertRVGuardAndReads(t *testing.T) {
	setupSyncDB(t)

	if err := upsertObjects([]*model.SyncObject{row("c1", "u1", "web", "default", "10")}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	// Older RV must not overwrite.
	if err := upsertObjects([]*model.SyncObject{row("c1", "u1", "web-old", "default", "9")}); err != nil {
		t.Fatalf("upsert old: %v", err)
	}
	objs, err := ListObjects("c1", "pods", "_all")
	if err != nil {
		t.Fatalf("ListObjects: %v", err)
	}
	if len(objs) != 1 {
		t.Fatalf("got %d objects, want 1", len(objs))
	}

	// Newer RV updates name.
	if err := upsertObjects([]*model.SyncObject{row("c1", "u1", "web-new", "default", "11")}); err != nil {
		t.Fatalf("upsert newer: %v", err)
	}
	raw, err := GetObject("c1", "pods", "default", "web-new")
	if err != nil {
		t.Fatalf("GetObject by new name: %v", err)
	}
	if raw == nil {
		t.Fatal("object missing after newer upsert")
	}

	// Namespace scoping.
	if err := upsertObjects([]*model.SyncObject{row("c1", "u2", "kube", "kube-system", "5")}); err != nil {
		t.Fatalf("upsert second: %v", err)
	}
	kubeObjs, err := ListObjects("c1", "pods", "kube-system")
	if err != nil || len(kubeObjs) != 1 {
		t.Fatalf("namespace filter: %v, %d", err, len(kubeObjs))
	}

	// Search matches lowercase substring.
	rows, err := SearchObjectRows("c1", "pods", "WEB", 10)
	if err != nil || len(rows) != 1 || rows[0].Name != "web-new" {
		t.Fatalf("search: %v, %d", err, len(rows))
	}

	// Other clusters are isolated.
	if err := upsertObjects([]*model.SyncObject{row("c2", "u3", "other", "default", "1")}); err != nil {
		t.Fatalf("upsert c2: %v", err)
	}
	only, err := ListObjects("c1", "pods", "_all")
	if err != nil || len(only) != 2 {
		t.Fatalf("cluster isolation: %v, %d", err, len(only))
	}
}

func TestSoftDeleteAndPrune(t *testing.T) {
	setupSyncDB(t)

	if err := upsertObjects([]*model.SyncObject{
		row("c1", "u1", "a", "default", "1"),
		row("c1", "u2", "b", "default", "1"),
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	softDeleteObject("c1", "u1")

	objs, err := ListObjects("c1", "pods", "_all")
	if err != nil || len(objs) != 1 {
		t.Fatalf("after soft delete: %v, %d", err, len(objs))
	}

	// pruneVanished soft-deletes rows missing from the fresh snapshot.
	if err := pruneVanished("c1", "pods", []*model.SyncObject{row("c1", "u3", "c", "default", "1")}); err != nil {
		t.Fatalf("prune: %v", err)
	}
	objs, err = ListObjects("c1", "pods", "_all")
	if err != nil || len(objs) != 0 {
		t.Fatalf("after prune: %v, %d", err, len(objs))
	}
}

func TestActiveFreshnessGating(t *testing.T) {
	setupSyncDB(t)

	if Active("c1", "pods") {
		t.Fatal("no task must mean inactive")
	}

	// Fresh RUNNING task → active.
	trackTask("c1", "pods", model.SyncTaskRunning, 5, "")
	if !Active("c1", "pods") {
		t.Fatal("fresh RUNNING task must be active")
	}

	// Stale task → inactive.
	stale := time.Now().Add(-2 * freshness)
	model.DB.Model(&model.SyncTask{}).
		Where("cluster_name = ? AND resource = ?", "c1", "pods").
		Update("last_sync_at", stale)
	if Active("c1", "pods") {
		t.Fatal("stale task must be inactive")
	}

	// ERROR task → inactive.
	trackTask("c1", "pods", model.SyncTaskError, 0, "boom")
	if Active("c1", "pods") {
		t.Fatal("error task must be inactive")
	}
}

func TestParseWhitelist(t *testing.T) {
	if got := parseWhitelist(""); len(got) != len(DefaultWhitelist) {
		t.Fatalf("empty whitelist must fall back to default, got %v", got)
	}
	got := parseWhitelist("Pods, pods , deployments,")
	if len(got) != 2 || got[0] != "pods" || got[1] != "deployments" {
		t.Fatalf("parseWhitelist = %v", got)
	}
}

func TestUpsertRevivesSoftDeleted(t *testing.T) {
	setupSyncDB(t)

	if err := upsertObjects([]*model.SyncObject{row("c1", "u1", "a", "default", "1")}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	softDeleteObject("c1", "u1")

	// Object recreated in the cluster with a newer RV: row must come back.
	if err := upsertObjects([]*model.SyncObject{row("c1", "u1", "a", "default", "2")}); err != nil {
		t.Fatalf("upsert revived: %v", err)
	}
	objs, err := ListObjects("c1", "pods", "_all")
	if err != nil || len(objs) != 1 {
		t.Fatalf("revived object missing: %v, %d", err, len(objs))
	}
}
