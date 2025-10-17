package config

import (
	"flag"

	"github.com/caarlos0/env"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Server     string `env:"RUN_ADDRESS" envDefault:"localhost:8080"`
	Database   string `env:"DATABASE_URI" envDefault:"postgres://postgres:qwer1234@localhost:5432/gofermart"`
	AccrualSys string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"http://localhost:8080/"`
	SecretKey  string `env:"SECRET_KEY" envDefault:"secret"`
}

func GetConfig(log *logrus.Logger) (*Config, error) {
	cfg := &Config{}

	if err := env.Parse(cfg); err != nil {
		return nil, err
	}

	flag.StringVar(&cfg.Server, "a", cfg.Server, "HTTP server address")
	flag.StringVar(&cfg.Database, "d", cfg.Database, "Database connections")
	flag.StringVar(&cfg.AccrualSys, "r", cfg.AccrualSys, "Accrual system")
	flag.StringVar(&cfg.SecretKey, "k", cfg.SecretKey, "Secret key")
	flag.Parse()

	log.WithFields(logrus.Fields{"cfg": cfg}).Info("Итоговая конфигурация")
	return cfg, nil
}
