package webrtc

import (
	"os"
	"strings"
)

func IsProduction() bool {
	env := os.Getenv("ENVIRONMENT")
	if env == "" {
		// Backward compatibility with existing misspelled variable.
		env = os.Getenv("ENVIROMENT")
	}
	return strings.EqualFold(env, "PRODUCTION")
}
