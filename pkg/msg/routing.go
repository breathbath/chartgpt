package msg

import "strings"

func MatchCommand(msg string, variants []string) bool {
	for _, v := range variants {
		if strings.ToLower(msg) == strings.ToLower(CommandPrefix+strings.TrimPrefix(v, CommandPrefix)) {
			return true
		}
	}

	return false
}

func IsCommand(msg string) bool {
	return strings.HasPrefix(msg, CommandPrefix)
}
