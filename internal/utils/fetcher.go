package utils

import (
	"errors"
	"net/url"
	"strings"
)

var Timezone string

func Init(timezone string) {

	Timezone = timezone

}

func NormalizeURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("empty url")
	}

	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)

	u.Path = strings.TrimRight(u.Path, "/")

	return u.String(), nil
}
