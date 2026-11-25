package config

import "os"

type Config struct {
	PORT              string
	CASSANDRA_HOST    string
	KEYSPACE          string
	USER_SERVICE_PORT string
}

func LoadConfigs() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}
	host := os.Getenv("CASSANDRA_HOSTS")
	if host == "" {
		host = "cassandra"
	}
	userPort := os.Getenv("USER_SERVICE_PORT")
	if userPort == "" {
		userPort = "8082" // Default user service port
	}

	return Config{
		PORT:              port,
		CASSANDRA_HOST:    host,
		KEYSPACE:          "filesify",
		USER_SERVICE_PORT: userPort,
	}
}
