package config

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env string `yaml:"env" env:"ENV" env-default:"local" env-required:"true"`

	Nats            Nats            `yaml:"nats"`
	PostgresStorage PostgresStorage `yaml:"postgres_storage"`
	RedisStorage    RedisStorage    `yaml:"redis_storage"`
	HTTPServer      HTTPServer      `yaml:"http_server"`
}

type Nats struct {
	Host     string `yaml:"host" env-default:"nats"`
	Port     string `yaml:"port" env-default:"4222"`
	User     string `yaml:"user" env-default:"hezzl_admin"`
	Password string `yaml:"password" env-default:"hezzl_password"`

	StreamName string `yaml:"stream_name" env-default:"CLICKHOUSE"`
	Subject    string `yaml:"subject" env-default:"clickhouse.logs"`
	//ConsumerName string `yaml:"consumer_name" env-default:"clickhouse-logs-consumer"`
	//
	//AckWait   time.Duration `yaml:"ack_wait" env-default:"30s"`
	//BatchSize int           `yaml:"batch_size" env-default:"10"`
}

type PostgresStorage struct {
	Host     string `yaml:"host" env-default:"postgres"`
	Port     string `yaml:"port" env-default:"5432"`
	Database string `yaml:"database" env-default:"postgres"`
	User     string `yaml:"user" env-default:"postgres"`
	Password string `yaml:"password" env-default:"postgres"`
}

type RedisStorage struct {
	Addr       string `yaml:"addr" env-default:"redis"`
	Password   string `yaml:"password" env-default:"password"`
	DB         int    `yaml:"db" env-default:"0"`
	MaxRetries int    `yaml:"max_retries" env-default:"3"`
}

type HTTPServer struct {
	Address     string        `yaml:"address" env:"ADDRESS" env-default:"localhost:8080"`
	Timeout     time.Duration `yaml:"timeout" env:"TIMEOUT" env-default:"4s"`
	IdleTimeout time.Duration `yaml:"idle_timeout" env:"IDLE_TIMEOUT" env-default:"60s"`
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

func (s *PostgresStorage) DSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s", s.User, s.Password, s.Host, s.Port, s.Database)
}

func (n *Nats) URL() string {
	return fmt.Sprintf("nats://%s:%s@%s:%s", n.User, n.Password, n.Host, n.Port)
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
