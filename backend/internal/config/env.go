package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	Address      string
	DatabaseURL  string
	CookieSecure bool
}

func Load() (Config, error) {
	c := Config{Address: os.Getenv("HTTP_ADDRESS"), DatabaseURL: os.Getenv("DATABASE_URL")}
	if c.Address == "" {
		c.Address = ":8080"
	}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL не задан")
	}
	if value := os.Getenv("COOKIE_SECURE"); value != "" {
		var err error
		c.CookieSecure, err = strconv.ParseBool(value)
		if err != nil {
			return c, fmt.Errorf("COOKIE_SECURE: %w", err)
		}
	}
	return c, nil
}
