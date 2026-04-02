package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Port         string `json:"port"`
	DatabaseURL  string `json:"database_url"`
	JWTSecret    string `json:"jwt_secret"`
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     string `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	SMTPFrom     string `json:"smtp_from"`
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("could not open config file: %w", err)
	}
	defer file.Close()

	cfg := &Config{}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not decode config file: %w", err)
	}

	// Set defaults if missing
	if cfg.Port == "" {
		cfg.Port = "8080"
	}

	return cfg, nil
}
