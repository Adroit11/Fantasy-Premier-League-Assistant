package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port         int
	FPLBaseURL   string
	FPLUserAgent string
	CacheTTL     time.Duration
	Environment  string

	// MySQL Configuration
	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBEnabled  bool
}

func (c *Config) MySQLDSN() string {
	// Format: username:password@tcp(host:port)/dbname?parseTime=true&charset=utf8mb4
	pwd := c.DBPassword
	if pwd != "" {
		pwd = ":" + pwd
	}
	return fmt.Sprintf("%s%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4&timeout=5s&readTimeout=5s&writeTimeout=5s&interpolateParams=true",
		c.DBUser, pwd, c.DBHost, c.DBPort, c.DBName)
}

func LoadConfig() *Config {
	// Try loading .env from current directory, parent directory, or root
	loadDotEnv(".env")
	loadDotEnv("../.env")
	loadDotEnv("../../.env")

	port := getEnvInt("PORT", getEnvInt("BACKEND_PORT", 18492))
	baseURL := getEnv("FPL_BASE_URL", "https://fantasy.premierleague.com/api")
	userAgent := getEnv("FPL_USER_AGENT", "FPL-Assistant-Desktop/1.0 (Go; Windows)")
	cacheTTLMin := getEnvInt("CACHE_TTL_MINUTES", 60)
	env := getEnv("APP_ENV", getEnv("ENVIRONMENT", "development"))

	// MySQL defaults
	dbHost := getEnv("DB_HOST", "127.0.0.1")
	dbPort := getEnvInt("DB_PORT", 3306)
	dbUser := getEnv("DB_USER", "root")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "fpl_assistant")
	dbEnabled := getEnvBool("DB_ENABLED", true)

	return &Config{
		Port:         port,
		FPLBaseURL:   baseURL,
		FPLUserAgent: userAgent,
		CacheTTL:     time.Duration(cacheTTLMin) * time.Minute,
		Environment:  env,
		DBHost:       dbHost,
		DBPort:       dbPort,
		DBUser:       dbUser,
		DBPassword:   dbPassword,
		DBName:       dbName,
		DBEnabled:    dbEnabled,
	}
}

func getEnv(key, defaultVal string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvBool(key string, defaultVal bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		lower := strings.ToLower(strings.TrimSpace(val))
		return lower == "true" || lower == "1" || lower == "yes"
	}
	return defaultVal
}

func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if _, exists := os.LookupEnv(key); !exists {
				_ = os.Setenv(key, val)
			}
		}
	}
}
