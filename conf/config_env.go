package conf

import (
	"github.com/caarlos0/env/v6"
)

// AppConfig presents app conf
type AppConfig struct {
	Port      string `env:"PORT" envDefault:"8081"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"text"`
	DBHost    string `env:"DB_HOST" envDefault:"10.10.1.4"`
	DBPort    string `env:"DB_PORT" envDefault:"5432"`
	DBUser    string `env:"DB_USER" envDefault:"finan_dev_user"`
	DBPass    string `env:"DB_PASS" envDefault:"Oo5Tah0re5eexoif"`
	DBName    string `env:"DB_NAME" envDefault:"finan_dev_ms_order_management"`
	EnableDB  string `env:"ENABLE_DB" envDefault:"true"`
}

var config AppConfig

func SetEnv() {
	_ = env.Parse(&config)
}

func LoadEnv() AppConfig {
	return config
}
