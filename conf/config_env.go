package conf

import (
	"github.com/caarlos0/env/v6"
)

// AppConfig presents app conf
type AppConfig struct {
	Port      string `env:"PORT" envDefault:"8000"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"text"`
	DBHost    string `env:"DB_HOST" envDefault:"localhost"`
	DBPort    string `env:"DB_PORT" envDefault:"5432"`
	DBUser    string `env:"DB_USER" envDefault:"root"`
	DBPass    string `env:"DB_PASS" envDefault:"password"`
	DBName    string `env:"DB_NAME" envDefault:"ms_user_tutorial"`
	EnableDB  string `env:"ENABLE_DB" envDefault:"true"`

	MSBusinessManagement string `env:"MS_BUSINESS_MANAGEMENT"  envDefault:"http://localhost:8012"`
}

var config AppConfig

func SetEnv() {
	_ = env.Parse(&config)
}

func LoadEnv() AppConfig {
	return config
}
