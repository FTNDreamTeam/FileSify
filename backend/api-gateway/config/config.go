package config

import "os"

type Config struct {
	gateway_address string
}

func GetConfig() Config {
	return Config{
		gateway_address: os.Getenv("GATEWAY_ADDRESS"),
	}
}
