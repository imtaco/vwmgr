package utils

import (
	"fmt"
	"net/url"
	"strings"
)

func PGURLtoGormDSN(pgURL string) (string, error) {
	u, err := url.Parse(pgURL)
	if err != nil {
		return "", err
	}

	user := u.User.Username()
	password, _ := u.User.Password()
	host := u.Hostname()
	port := u.Port()
	dbname := strings.TrimPrefix(u.Path, "/")

	if port == "" {
		port = "5432"
	}

	// Default to sslmode=disable if not provided
	sslmode := "disable"
	timezone := "UTC"

	query := u.Query()
	if s := query.Get("sslmode"); s != "" {
		sslmode = s
	}
	if tz := query.Get("TimeZone"); tz != "" {
		timezone = tz
	}

	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		host, user, password, dbname, port, sslmode, timezone,
	), nil
}
