package common

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestValidateEncryptionKey(t *testing.T) {
	oldKey := EncryptKey
	oldAllow := AllowInsecureDefaults
	defer func() {
		EncryptKey = oldKey
		AllowInsecureDefaults = oldAllow
	}()

	EncryptKey = "too-short"
	AllowInsecureDefaults = false
	if err := ValidateEncryptionKey(); err == nil {
		t.Fatal("ValidateEncryptionKey() accepted a short key")
	}

	EncryptKey = "kube-osprey-default-encryption-key-change-in-production"
	if err := ValidateEncryptionKey(); err == nil {
		t.Fatal("ValidateEncryptionKey() accepted the public default key")
	}

	AllowInsecureDefaults = true
	if err := ValidateEncryptionKey(); err != nil {
		t.Fatalf("ValidateEncryptionKey() with explicit development override = %v", err)
	}

	EncryptKey = "a-strong-development-encryption-key-123456"
	AllowInsecureDefaults = false
	if err := ValidateEncryptionKey(); err != nil {
		t.Fatalf("ValidateEncryptionKey() = %v", err)
	}
}

func TestValidateWebSocketOrigin(t *testing.T) {
	oldOrigins := CORSAllowedOrigins
	defer func() { CORSAllowedOrigins = oldOrigins }()
	CORSAllowedOrigins = nil

	req := httptest.NewRequest("GET", "https://kube-osprey.example.com/socket", nil)
	req.Host = "kube-osprey.example.com"
	req.Header.Set("Origin", "https://kube-osprey.example.com")
	if !ValidateWebSocketOrigin(req) {
		t.Fatal("same-origin WebSocket request was rejected")
	}

	req.Header.Set("Origin", "https://evil.example.com")
	if ValidateWebSocketOrigin(req) {
		t.Fatal("cross-origin WebSocket request was accepted")
	}

	CORSAllowedOrigins = []string{"https://admin.example.com"}
	req.Header.Set("Origin", "https://admin.example.com")
	if !ValidateWebSocketOrigin(req) {
		t.Fatal("explicitly allowed WebSocket origin was rejected")
	}
}

func TestLoadEnvs(t *testing.T) {
	old := struct {
		JwtSecret             string
		Port                  string
		AgentPodNamespace     string
		NodeTerminalImage     string
		KubectlTerminalImage  string
		DBDSN                 string
		DBType                string
		EncryptKey            string
		AllowInsecureDefaults bool
		AnonymousUserEnabled  bool
		Host                  string
		DisableGZIP           bool
		EnableVersionCheck    bool
		Base                  string
		CORSAllowedOrigins    []string
		TrustedProxies        []string
	}{
		JwtSecret:             JwtSecret,
		Port:                  Port,
		AgentPodNamespace:     AgentPodNamespace,
		NodeTerminalImage:     NodeTerminalImage,
		KubectlTerminalImage:  KubectlTerminalImage,
		DBDSN:                 DBDSN,
		DBType:                DBType,
		EncryptKey:            EncryptKey,
		AllowInsecureDefaults: AllowInsecureDefaults,
		AnonymousUserEnabled:  AnonymousUserEnabled,
		Host:                  Host,
		DisableGZIP:           DisableGZIP,
		EnableVersionCheck:    EnableVersionCheck,
		Base:                  Base,
		CORSAllowedOrigins:    append([]string(nil), CORSAllowedOrigins...),
		TrustedProxies:        append([]string(nil), TrustedProxies...),
	}
	defer func() {
		JwtSecret = old.JwtSecret
		Port = old.Port
		AgentPodNamespace = old.AgentPodNamespace
		NodeTerminalImage = old.NodeTerminalImage
		KubectlTerminalImage = old.KubectlTerminalImage
		DBDSN = old.DBDSN
		DBType = old.DBType
		EncryptKey = old.EncryptKey
		AllowInsecureDefaults = old.AllowInsecureDefaults
		AnonymousUserEnabled = old.AnonymousUserEnabled
		Host = old.Host
		DisableGZIP = old.DisableGZIP
		EnableVersionCheck = old.EnableVersionCheck
		Base = old.Base
		CORSAllowedOrigins = append([]string(nil), old.CORSAllowedOrigins...)
		TrustedProxies = append([]string(nil), old.TrustedProxies...)
	}()

	CORSAllowedOrigins = nil
	TrustedProxies = nil

	t.Setenv("JWT_SECRET", "test-jwt-secret")
	t.Setenv("PORT", "9090")
	t.Setenv("NAMESPACE", "test-namespace")
	t.Setenv("NODE_TERMINAL_IMAGE", "test-node-image")
	t.Setenv("KUBECTL_TERMINAL_IMAGE", "test-kubectl-image")
	t.Setenv("CLUSTER_AGENT_IMAGE", "test-cluster-agent-image")
	t.Setenv("DB_DSN", "test.db")
	t.Setenv("DB_TYPE", "mysql")
	t.Setenv("KUBE_OSPREY_ENCRYPT_KEY", "test-encrypt-key")
	t.Setenv("ANONYMOUS_USER_ENABLED", "true")
	t.Setenv("HOST", "example.com")
	t.Setenv("DISABLE_GZIP", "false")
	t.Setenv("DISABLE_VERSION_CHECK", "true")
	t.Setenv("KUBE_OSPREY_BASE", "kube-osprey/")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://example.com ,,")
	t.Setenv("TRUSTED_PROXIES", "10.42.0.0/16, 192.0.2.10 ,, ")

	LoadEnvs()

	if JwtSecret != "test-jwt-secret" {
		t.Fatalf("JwtSecret = %q, want %q", JwtSecret, "test-jwt-secret")
	}
	if Port != "9090" {
		t.Fatalf("Port = %q, want %q", Port, "9090")
	}
	if AgentPodNamespace != "test-namespace" {
		t.Fatalf("AgentPodNamespace = %q, want %q", AgentPodNamespace, "test-namespace")
	}
	if NodeTerminalImage != "test-node-image" {
		t.Fatalf("NodeTerminalImage = %q, want %q", NodeTerminalImage, "test-node-image")
	}
	if KubectlTerminalImage != "test-kubectl-image" {
		t.Fatalf("KubectlTerminalImage = %q, want %q", KubectlTerminalImage, "test-kubectl-image")
	}
	if DBDSN != "test.db" {
		t.Fatalf("DBDSN = %q, want %q", DBDSN, "test.db")
	}
	if DBType != "mysql" {
		t.Fatalf("DBType = %q, want %q", DBType, "mysql")
	}
	if EncryptKey != "test-encrypt-key" {
		t.Fatalf("EncryptKey = %q, want %q", EncryptKey, "test-encrypt-key")
	}
	if !AnonymousUserEnabled {
		t.Fatalf("AnonymousUserEnabled = %v, want true", AnonymousUserEnabled)
	}
	if Host != "example.com" {
		t.Fatalf("Host = %q, want %q", Host, "example.com")
	}
	if DisableGZIP {
		t.Fatalf("DisableGZIP = %v, want false", DisableGZIP)
	}
	if EnableVersionCheck {
		t.Fatalf("EnableVersionCheck = %v, want false", EnableVersionCheck)
	}
	if Base != "/kube-osprey" {
		t.Fatalf("Base = %q, want %q", Base, "/kube-osprey")
	}

	wantOrigins := []string{"http://localhost:5173", "https://example.com"}
	if !reflect.DeepEqual(CORSAllowedOrigins, wantOrigins) {
		t.Fatalf("CORSAllowedOrigins = %#v, want %#v", CORSAllowedOrigins, wantOrigins)
	}

	wantTrustedProxies := []string{"10.42.0.0/16", "192.0.2.10"}
	if !reflect.DeepEqual(TrustedProxies, wantTrustedProxies) {
		t.Fatalf("TrustedProxies = %#v, want %#v", TrustedProxies, wantTrustedProxies)
	}
}

func TestLoadEnvs_BaseAlreadyHasLeadingSlash(t *testing.T) {
	old := struct {
		EncryptKey         string
		Base               string
		CORSAllowedOrigins []string
	}{
		EncryptKey:         EncryptKey,
		Base:               Base,
		CORSAllowedOrigins: append([]string(nil), CORSAllowedOrigins...),
	}
	defer func() {
		EncryptKey = old.EncryptKey
		Base = old.Base
		CORSAllowedOrigins = append([]string(nil), old.CORSAllowedOrigins...)
	}()

	t.Setenv("KUBE_OSPREY_ENCRYPT_KEY", "test-encrypt-key")
	t.Setenv("KUBE_OSPREY_BASE", "/kube-osprey/")

	LoadEnvs()

	if Base != "/kube-osprey" {
		t.Fatalf("Base = %q, want %q", Base, "/kube-osprey")
	}
}
