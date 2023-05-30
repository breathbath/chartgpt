package storage

import "strings"

// GenerateCacheKey allows to generate consistent keys with a small probability of conflicts
/**
version: model version which is stored under this key, if model has inconsistent changes, the version can be increased, example v1, v2, v3
platform: e.g. "chatgpt, bard" etc
domain: "user, conversation, setting"
uniqueParts: "123 as user id or 3456 as conversation id"
*/
func GenerateCacheKey(version, platform, domain string, uniqueParts ...string) string {
	parts := []string{
		version,
		strings.ToLower(platform),
		strings.ToLower(domain),
	}

	parts = append(parts, uniqueParts...)

	return strings.Join(parts, "/")
}
