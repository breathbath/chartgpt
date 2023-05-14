package utils

import (
	"fmt"
	"regexp"
	"strings"
)

func ExtractCommandValue(rawMsg, command string) string {
	r := regexp.MustCompile(fmt.Sprintf(`^%s($|\s.*)`, command))
	foundResults := r.FindStringSubmatch(rawMsg)
	if len(foundResults) == 0 {
		return ""
	}

	return strings.TrimSpace(foundResults[1])
}
