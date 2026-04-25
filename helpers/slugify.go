package helpers

import (
	"regexp"
	"strings"
)

func ResourceName(parts ...string) string {
	slugged := make([]string, len(parts))
	for i, p := range parts {
		slugged[i] = Slugify(p)
	}
	return strings.Join(slugged, ".")
}

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9_-]+`)

// Slugify converts a string to a lowercase slug suitable for Pulumi resource names.
// It replaces non-alphanumeric characters (except hyphens and underscores) with underscores,
// then converts the result to lowercase.
//
// Examples:
//
//	Slugify("My-Org Name")     -> "my-org_name"
//	Slugify("oddbit.com")      -> "oddbit_com"
//	Slugify("user@example")    -> "user_example"
func Slugify(v string) string {
	return nonAlphanumeric.ReplaceAllString(strings.ToLower(v), "_")
}
