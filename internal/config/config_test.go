// config/config_test.go
package config

import (
	"flag"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Сохраняем оригинальные значения флагов
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	tests := []struct {
		name           string
		envVars        map[string]string
		args           []string
		expectedConfig *Config
		expectError    bool
	}{
		{
			name: "all env vars set",
			envVars: map[string]string{
				"RUN_ADDRESS":            ":8081",
				"DATABASE_URI":           "postgres://user:pass@localhost:5432/testdb",
				"ACCRUAL_SYSTEM_ADDRESS": "http://localhost:8082",
			},
			args: []string{},
			expectedConfig: &Config{
				RunAddress:           ":8081",
				DatabaseURI:          "postgres://user:pass@localhost:5432/testdb",
				AccrualSystemAddress: "http://localhost:8082",
			},
			expectError: false,
		},
		{
			name:    "default values",
			envVars: map[string]string{},
			args:    []string{},
			expectedConfig: &Config{
				RunAddress:           ":8080",
				DatabaseURI:          "postgres://postgres:0000@localhost:5432/gophermart?sslmode=disable",
				AccrualSystemAddress: "http://localhost:8081",
			},
			expectError: false,
		},
		{
			name:    "command line args",
			envVars: map[string]string{},
			args:    []string{"-a", ":8082", "-d", "postgres://test:test@localhost:5432/testdb"},
			expectedConfig: &Config{
				RunAddress:           ":8082",
				DatabaseURI:          "postgres://test:test@localhost:5432/testdb",
				AccrualSystemAddress: "http://localhost:8081",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Очищаем переменные окружения
			for key := range map[string]string{
				"RUN_ADDRESS":            "",
				"DATABASE_URI":           "",
				"ACCRUAL_SYSTEM_ADDRESS": "",
			} {
				os.Unsetenv(key)
			}

			// Устанавливаем тестовые значения
			for key, value := range tt.envVars {
				require.NoError(t, os.Setenv(key, value))
			}

			// Устанавливаем аргументы командной строки
			if len(tt.args) > 0 {
				os.Args = append([]string{"cmd"}, tt.args...)
			} else {
				os.Args = []string{"cmd"}
			}

			// Сбрасываем флаги перед каждым тестом
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			config, err := LoadConfig()

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedConfig.RunAddress, config.RunAddress)
			assert.Equal(t, tt.expectedConfig.DatabaseURI, config.DatabaseURI)
			assert.Equal(t, tt.expectedConfig.AccrualSystemAddress, config.AccrualSystemAddress)
		})
	}
}

func TestConfig_String(t *testing.T) {
	cfg := &Config{
		RunAddress:           ":8080",
		DatabaseURI:          "postgres://user:pass@localhost:5432/db",
		AccrualSystemAddress: "http://localhost:8081",
	}

	expected := "Config{RunAddress: :8080, DatabaseURI: postgres://user:pass@localhost:5432/db, AccrualSystemAddress: http://localhost:8081}"
	assert.Equal(t, expected, cfg.String())
}
