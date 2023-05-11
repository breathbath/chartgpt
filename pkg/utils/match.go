package utils

import "strings"

func MatchesAny(msg, prefix string, variants []string) bool {
	for _, v := range variants {
		prefixVariant := strings.ToLower(prefix + strings.TrimPrefix(v, prefix))
		if strings.HasPrefix(strings.ToLower(msg), prefixVariant) {
			return true
		}
	}

	return false
}
