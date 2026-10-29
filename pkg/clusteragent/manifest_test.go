package clusteragent

import (
	"strings"
	"testing"
)

func TestGenerateManifestPermissionProfiles(t *testing.T) {
	readOnly := GenerateManifestWithPermission("https://kube-osprey.example.com", "token", "key", "image:v1", "readOnly", "")
	if !strings.Contains(readOnly, "name: kube-osprey-cluster-agent-readonly") || strings.Contains(readOnly, "name: cluster-admin") {
		t.Fatalf("readOnly manifest contains unexpected role: %s", readOnly)
	}

	operator := GenerateManifestWithPermission("https://kube-osprey.example.com", "token", "key", "image:v1", "operator", "")
	if !strings.Contains(operator, "name: kube-osprey-cluster-agent-operator") || strings.Contains(operator, "name: cluster-admin") {
		t.Fatalf("operator manifest contains unexpected role: %s", operator)
	}

	custom := GenerateManifestWithPermission("https://kube-osprey.example.com", "token", "key", "image:v1", "custom", "team-osprey-reader")
	if !strings.Contains(custom, "name: team-osprey-reader") || strings.Contains(custom, "kind: ClusterRole\nmetadata:\n  name: team-osprey-reader") {
		t.Fatalf("custom manifest should bind an existing role: %s", custom)
	}
}

func TestGenerateManifest(t *testing.T) {
	manifest := GenerateManifest("https://kube-osprey.example.com", "my-token", "public-key", "ghcr.io/fadewalk/kube-osprey:v1.0")

	checks := []string{
		"apiVersion: v1\nkind: Secret",
		"name: kube-osprey-cluster-agent-token",
		"namespace: kube-system",
		"token: \"my-token\"",
		"apiVersion: v1\nkind: ServiceAccount",
		"name: kube-osprey-cluster-agent",
		"apiVersion: rbac.authorization.k8s.io/v1\nkind: ClusterRoleBinding",
		"name: cluster-admin",
		"apiVersion: apps/v1\nkind: Deployment",
		`image: "ghcr.io/fadewalk/kube-osprey:v1.0"`,
		"--server=$(KUBE_OSPREY_SERVER)",
		"--token=$(CLUSTER_AGENT_TOKEN)",
		"value: \"https://kube-osprey.example.com\"",
	}
	for _, want := range checks {
		if !strings.Contains(manifest, want) {
			t.Errorf("manifest missing %q\n--- manifest ---\n%s", want, manifest)
		}
	}

	// Should contain exactly 4 documents separated by ---
	if strings.Count(manifest, "---") != 3 {
		t.Errorf("expected 3 document separators, got %d", strings.Count(manifest, "---"))
	}
}

func TestGenerateManifestWhitespaceTrimmed(t *testing.T) {
	manifest := GenerateManifest("  https://kube-osprey.example.com  ", "  tok  ", "  public-key  ", "  img:tag  ")
	if !strings.Contains(manifest, `"https://kube-osprey.example.com"`) {
		t.Errorf("serverURL not trimmed:\n%s", manifest)
	}
	if !strings.Contains(manifest, `"tok"`) {
		t.Errorf("token not trimmed:\n%s", manifest)
	}
	if !strings.Contains(manifest, `image: "img:tag"`) {
		t.Errorf("image not trimmed:\n%s", manifest)
	}
}

func TestGenerateManifestSpecialCharacters(t *testing.T) {
	// Token with special characters that would break YAML if not quoted.
	manifest := GenerateManifest("https://kube-osprey.example.com", "tok:en\"'", "public-key", "img")
	if !strings.Contains(manifest, `"tok:en\"'"`) {
		t.Errorf("special characters in token not properly escaped:\n%s", manifest)
	}
}
