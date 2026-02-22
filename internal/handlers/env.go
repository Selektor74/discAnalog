package handlers

import (
	"os"
	"strings"
)

func isProduction() bool {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		env = os.Getenv("ENVIROMENT")
	}
	return strings.EqualFold(env, "PRODUCTION")
}

func turnHost() string {
	host := strings.TrimSpace(os.Getenv("TURN_HOST"))
	if host != "" {
		return host
	}
	return strings.TrimSpace(os.Getenv("TURN_PUBLIC_IP"))
}

func turnPort() string {
	port := strings.TrimSpace(os.Getenv("TURN_PORT"))
	if port == "" {
		return "3478"
	}
	return port
}

func turnCredentials() (string, string) {
	username := strings.TrimSpace(os.Getenv("TURN_USERNAME"))
	password := strings.TrimSpace(os.Getenv("TURN_PASSWORD"))

	if username != "" && password != "" {
		return username, password
	}

	raw := strings.TrimSpace(os.Getenv("TURN_USERS"))
	if raw == "" {
		return username, password
	}
	first := strings.Split(raw, ",")[0]
	parts := strings.SplitN(strings.TrimSpace(first), "=", 2)
	if len(parts) != 2 {
		return username, password
	}

	if username == "" {
		username = strings.TrimSpace(parts[0])
	}
	if password == "" {
		password = strings.TrimSpace(parts[1])
	}
	return username, password
}
