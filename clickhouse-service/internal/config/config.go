package config

import (
	"flag"
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
	"log"
	"os"
	"time"
)

type Config struct {
	Env string `yaml:"env" env:"ENV" env-default:"local" env-required:"true"`

	ClickHouseStorage ClickHouseStorage `yaml:"clickhouse"`
	Nats              Nats              `yaml:"nats"`
}

type Nats struct {
	Host     string `yaml:"host" env-default:"nats"`
	Port     string `yaml:"port" env-default:"4222"`
	User     string `yaml:"user" env-default:"hezzl_admin"`
	Password string `yaml:"password" env-default:"hezzl_password"`

	StreamName   string `yaml:"stream_name" env-default:"CLICKHOUSE"`
	Subject      string `yaml:"subject" env-default:"clickhouse.logs"`
	ConsumerName string `yaml:"consumer_name" env-default:"clickhouse-logs-consumer"`

	AckWait   time.Duration `yaml:"ack_wait" env-default:"30s"`
	BatchSize int           `yaml:"batch_size" env-default:"10"`
}

type ClickHouseStorage struct {
	Addr       string        `yaml:"addr" env-default:"clickhouse"`
	User       string        `yaml:"user" env-default:"hezzl_admin"`
	Password   string        `yaml:"password" env-default:"hezzl_password"`
	DB         string        `yaml:"db" env-default:"hezzl"`
	BatchTimer time.Duration `yaml:"batch_timer" env-default:"30s"`
}

func (c *ClickHouseStorage) DSN() string {
	return fmt.Sprintf("clickhouse://%s:%s@%s/%s", c.User, c.Password, c.Addr, c.DB)
}

func (n *Nats) URL() string {
	return fmt.Sprintf("nats://%s:%s@%s:%s", n.User, n.Password, n.Host, n.Port)
}

func MustLoad() *Config {
	configPath := fetchConfigPath()
	if configPath == "" {
		log.Fatal("CONFIG_PATH is required")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("config file not found: %s", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("failed to read config: %v", err)
	}

	return &cfg
}

// fetchConfigPath fetches config path from command line flag or environment variable.
// Priority: flag > env > default.
// Default value is empty string.
func fetchConfigPath() string {
	var res string

	flag.StringVar(&res, "config", "", "path to config file")
	flag.Parse()

	if res == "" {
		res = os.Getenv("CONFIG_PATH")
	}

	if res == "" {
		res = "./config/local.yaml"
	}

	return res
}
