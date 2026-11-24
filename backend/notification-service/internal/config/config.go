package config

import "os"

type Config struct {
	PORT           string
	CASSANDRA_HOST string
	KEYSPACE       string
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

	return Config{
		PORT:           port,
		CASSANDRA_HOST: host,
		KEYSPACE:       "filesify",
	}
}
