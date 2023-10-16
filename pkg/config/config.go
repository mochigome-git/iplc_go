package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from multiple .env.local files in a specific order
func LoadEnv(files ...string) {
	for _, file := range files {
		err := godotenv.Load(file)
		if err != nil {
			log.Fatalf("Error loading %s file: %v", file, err)
		}
	}
}

// GetEnvAsInt gets the value of an environment variable as a uint16
func GetEnvAsInt(name string, defaultValue int) int {
	if value, exists := os.LookupEnv(name); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
