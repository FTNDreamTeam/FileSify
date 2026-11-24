package config

import "os"

type Config struct {
	USER_SERVICE_PORT string
	GATEWAY_PORT      string
	STORAGE_PORT      string
	NOTIFICATION_PORT string
}

func LoadConfigs() Config {
	return Config{
		USER_SERVICE_PORT: os.Getenv("USER_SERVICE_PORT"),
		GATEWAY_PORT:      os.Getenv("GATEWAY_PORT"),
		STORAGE_PORT:      os.Getenv("STORAGE_PORT"),
		NOTIFICATION_PORT: os.Getenv("NOTIFICATION_PORT"),
	}
}
