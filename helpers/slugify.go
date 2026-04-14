package helpers

import (
	"regexp"
	"strings"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9_-]+`)

func Slugify(v string) string {
	return nonAlphanumeric.ReplaceAllString(strings.ToLower(v), "_")
}
