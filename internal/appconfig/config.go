package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultAddr = "127.0.0.1:8090"
)

type Config struct {
	Addr       string
	BenchRoot  string
	SiteName   string
	SiteConfig string
	DBHost     string
	DBPort     int
	DBName     string
	DBUser     string
	DBPassword string
}

type commonSiteConfig struct {
	DefaultSite string `json:"default_site"`
}

type siteConfig struct {
	DBHost     string `json:"db_host"`
	DBPort     int    `json:"db_port"`
	DBName     string `json:"db_name"`
	DBPassword string `json:"db_password"`
}

func LoadFromEnv() (Config, error) {
	benchRoot := strings.TrimSpace(os.Getenv("ERP_BENCH_ROOT"))
	if benchRoot == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return Config{}, fmt.Errorf("getwd: %w", err)
		}
		benchRoot = cwd
	}

	cfg := Config{
		Addr:      envOrDefault("ERP_READ_ADDR", defaultAddr),
		BenchRoot: benchRoot,
	}

	siteName := strings.TrimSpace(os.Getenv("ERP_SITE_NAME"))
	if siteName == "" {
		commonPath := filepath.Join(benchRoot, "sites", "common_site_config.json")
		commonCfg, err := loadCommonSiteConfig(commonPath)
		if err != nil {
			return Config{}, err
		}
		siteName = strings.TrimSpace(commonCfg.DefaultSite)
	}
	if siteName == "" {
		return Config{}, fmt.Errorf("site name is empty")
	}
	cfg.SiteName = siteName

	cfg.SiteConfig = strings.TrimSpace(os.Getenv("ERP_SITE_CONFIG"))
	if cfg.SiteConfig == "" {
		cfg.SiteConfig = filepath.Join(benchRoot, "sites", siteName, "site_config.json")
	}

	siteCfg, err := loadSiteConfig(cfg.SiteConfig)
	if err != nil {
		return Config{}, err
	}

	cfg.DBHost = strings.TrimSpace(siteCfg.DBHost)
	if cfg.DBHost == "" {
		cfg.DBHost = "127.0.0.1"
	}
	cfg.DBPort = siteCfg.DBPort
	if cfg.DBPort == 0 {
		cfg.DBPort = 3306
	}
	cfg.DBName = strings.TrimSpace(siteCfg.DBName)
	cfg.DBPassword = strings.TrimSpace(siteCfg.DBPassword)
	cfg.DBUser = envOrDefault("ERP_DB_USER", cfg.DBName)

	if cfg.DBName == "" {
		return Config{}, fmt.Errorf("db_name is empty in %s", cfg.SiteConfig)
	}
	if cfg.DBPassword == "" {
		return Config{}, fmt.Errorf("db_password is empty in %s", cfg.SiteConfig)
	}

	if portRaw := strings.TrimSpace(os.Getenv("ERP_DB_PORT")); portRaw != "" {
		port, err := strconv.Atoi(portRaw)
		if err != nil {
			return Config{}, fmt.Errorf("invalid ERP_DB_PORT: %w", err)
		}
		cfg.DBPort = port
	}
	if host := strings.TrimSpace(os.Getenv("ERP_DB_HOST")); host != "" {
		cfg.DBHost = host
	}

	return cfg, nil
}

func (c Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&collation=utf8mb4_unicode_ci",
		c.DBUser,
		c.DBPassword,
		c.DBHost,
		c.DBPort,
		c.DBName,
	)
}

func loadCommonSiteConfig(path string) (commonSiteConfig, error) {
	var cfg commonSiteConfig
	if err := loadJSON(path, &cfg); err != nil {
		return commonSiteConfig{}, fmt.Errorf("load common site config: %w", err)
	}
	return cfg, nil
}

func loadSiteConfig(path string) (siteConfig, error) {
	var cfg siteConfig
	if err := loadJSON(path, &cfg); err != nil {
		return siteConfig{}, fmt.Errorf("load site config: %w", err)
	}
	return cfg, nil
}

func loadJSON(path string, dst any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func envOrDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
