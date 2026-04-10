package appconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("ERP_BENCH_ROOT", t.TempDir())

	benchRoot := os.Getenv("ERP_BENCH_ROOT")
	commonPath := filepath.Join(benchRoot, "sites", "common_site_config.json")
	sitePath := filepath.Join(benchRoot, "sites", "erp.localhost", "site_config.json")

	if err := os.MkdirAll(filepath.Dir(sitePath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := os.WriteFile(commonPath, []byte(`{"default_site":"erp.localhost"}`), 0o644); err != nil {
		t.Fatalf("write common config: %v", err)
	}
	if err := os.WriteFile(sitePath, []byte(`{"db_name":"erpdb","db_password":"secret"}`), 0o644); err != nil {
		t.Fatalf("write site config: %v", err)
	}

	cfg, err := LoadFromEnv()
	if err != nil {
		t.Fatalf("LoadFromEnv error: %v", err)
	}

	if cfg.SiteName != "erp.localhost" {
		t.Fatalf("site name = %q", cfg.SiteName)
	}
	if cfg.DBHost != "127.0.0.1" {
		t.Fatalf("db host = %q", cfg.DBHost)
	}
	if cfg.DBPort != 3306 {
		t.Fatalf("db port = %d", cfg.DBPort)
	}
	if cfg.DBUser != "erpdb" {
		t.Fatalf("db user = %q", cfg.DBUser)
	}
}
