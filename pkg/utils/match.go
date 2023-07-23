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

func MatchesCommands(msg string, commands []string) bool {
	for _, command := range commands {
		if MatchesCommand(msg, command) {
			return true
		}
	}

	return false
}

func MatchesCommand(msg, command string) bool {
	command = strings.TrimPrefix(command, "/")
	re := regexp.MustCompile(fmt.Sprintf(`^/\b%s\b`, command))
	return re.MatchString(msg)
}
