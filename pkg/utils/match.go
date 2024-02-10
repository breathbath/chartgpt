package utils

import (
	"fmt"
	"math/rand"
	"regexp"
	"strings"
	"time"
)

func ExtractCommandValue(rawMsg, command string) string {
	r := regexp.MustCompile(fmt.Sprintf(`(?s)^%s($|\s.*)`, command))
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

func SelectRandomMessage(messages []string) string {
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(messages))

	randomMessage := messages[randomIndex]
	return randomMessage
}
