package conf

import (
	"github.com/caarlos0/env/v6"
)

// AppConfig presents app conf
type AppConfig struct {
	Port      string `env:"PORT" envDefault:"8000"`
	LogFormat string `env:"LOG_FORMAT" envDefault:"text"`
	DBHost    string `env:"DB_HOST" envDefault:"dbmasternode.stg.int.finan.cc"`
	DBPort    string `env:"DB_PORT" envDefault:"5432"`
	DBUser    string `env:"DB_USER" envDefault:"finan_dev_user"`
	DBPass    string `env:"DB_PASS" envDefault:"Oo5Tah0re5eexoif"`
	DBName    string `env:"DB_NAME" envDefault:"finan_dev_ms_order_management"`
	EnableDB  string `env:"ENABLE_DB" envDefault:"true"`

	MSBusinessManagement     string `env:"MS_BUSINESS_MANAGEMENT"  envDefault:"http://localhost:8012"`
	MSProductManagement      string `env:"MS_PRODUCT_MANAGEMENT" envDefault:"http://localhost:8094"`
	FinanProduct             string `env:"FINAN_PRODUCT" envDefault:"http://localhost:8093"`
	MSUserManagement         string `env:"MS_USER_MANAGEMENT"  envDefault:"http://127.0.0.1:8088"`
	MSPromotionManagement    string `env:"MS_PROMOTION_MANAGEMENT" envDefault:"http://localhost:8083"`
	MSConsumer               string `env:"MS_CONSUMER" envDefault:"http://127.0.0.1:8011"`
	MSTransactionManagement  string `env:"MS_TRANSACTION_MANAGEMENT" envDefault:"http://localhost:8084"`
	FinanTransaction         string `env:"FINAN_TRANSACTION" envDefault:"http://localhost:8084"`
	MSChat                   string `env:"MS_CHAT" envDefault:"http://ms-chat"`
	MSNotificationManagement string `env:"MS_NOTIFICATION_MANAGEMENT" envDefault:"http://localhost:8083"`
	MSMediaManagement        string `env:"MS_MEDIA_MANAGEMENT" envDefault:"http://localhost:8082"`
	MSWarehouseManagement    string `env:"MS_WAREHOUSE_MANAGEMENT" envDefault:"http://localhost:8888"`
	ApiKeySendinblue         string `env:"API_KEY_SENDINBLUE" envDefault:"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"`
	BIServerBaseURL          string `env:"BI_SERVER_BASE_URL" envDefault:"http://122.248.233.230:8001"`
	BIServerToken            string `env:"BI_SERVER_TOKEN" envDefault:"1c26WtGMKNecLzX5BBea-7kYQDQo7XVm3JyeFkKEf-pdpEJtBwiWPhIdWxHupO"`
}

var config AppConfig

func SetEnv() {
	_ = env.Parse(&config)
}

func LoadEnv() AppConfig {
	return config
}
