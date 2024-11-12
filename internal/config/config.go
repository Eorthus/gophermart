// config/config.go
package config

import (
	"flag"
	"fmt"
	"os"
)

// Config содержит конфигурационные параметры приложения
type Config struct {
	// Адрес и порт запуска сервиса
	RunAddress string `env:"RUN_ADDRESS" envDefault:":8080"`

	// Адрес подключения к базе данных
	DatabaseURI string `env:"DATABASE_URI" envDefault:"postgres://postgres:0000@localhost:5432/gophermart?sslmode=disable"`

	// Адрес системы расчёта начислений
	AccrualSystemAddress string `env:"ACCRUAL_SYSTEM_ADDRESS" envDefault:"http://localhost:8081"`
}

// LoadConfig загружает конфигурацию из переменных окружения и флагов
func LoadConfig() (*Config, error) {
	var cfg Config

	// Определяем флаги командной строки
	flag.StringVar(&cfg.RunAddress, "a", "", "Address and port to run server")
	flag.StringVar(&cfg.DatabaseURI, "d", "", "Database URI")
	flag.StringVar(&cfg.AccrualSystemAddress, "r", "http://localhost:8081", "Accrual system address")

	// Парсим флаги
	flag.Parse()

	// Приоритет отдаём переменным окружения
	if envRunAddr := os.Getenv("RUN_ADDRESS"); envRunAddr != "" {
		cfg.RunAddress = envRunAddr
	}

	if envDBURI := os.Getenv("DATABASE_URI"); envDBURI != "" {
		cfg.DatabaseURI = envDBURI
	}

	if envAccrualAddr := os.Getenv("ACCRUAL_SYSTEM_ADDRESS"); envAccrualAddr != "" {
		cfg.AccrualSystemAddress = envAccrualAddr
	}

	// Устанавливаем значения по умолчанию, если не заданы ни флаги, ни переменные окружения
	if cfg.RunAddress == "" {
		cfg.RunAddress = ":8080"
	}

	if cfg.DatabaseURI == "" {
		cfg.DatabaseURI = "postgres://postgres:0000@localhost:5432/gophermart?sslmode=disable"
	}

	if cfg.AccrualSystemAddress == "" {
		return nil, fmt.Errorf("accrual system address is not set")
	}

	return &cfg, nil
}

// String возвращает строковое представление конфигурации (для логгирования)
func (c *Config) String() string {
	return "Config{" +
		"RunAddress: " + c.RunAddress +
		", DatabaseURI: " + maskPassword(c.DatabaseURI) +
		", AccrualSystemAddress: " + c.AccrualSystemAddress +
		"}"
}

// maskPassword маскирует пароль в строке подключения к БД для безопасного логгирования
func maskPassword(dbURI string) string {
	return dbURI // TODO: implement password masking
}
