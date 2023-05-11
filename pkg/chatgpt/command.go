package chatgpt

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"context"
	"fmt"
	"regexp"
	"strings"
)

const CommandPrefix string = "@"

type CommandHandler interface {
	CanHandle(ctx context.Context, req *msg.Request) bool
	Handle(ctx context.Context, req *msg.Request) (*msg.Response, error)
	GetHelp() string
}

type Commands []CommandHandler

func (cc Commands) GetHelp() string {
	commandHelps := make([]string, len(cc))
	for i, c := range cc {
		commandHelps[i] = c.GetHelp()
	}

	return strings.Join(commandHelps, "\n")
}

func BuildCommandHandlers(db storage.Client, cfg *Config, loader *Loader) Commands {
	return []CommandHandler{
		NewSetModelCommand(cfg, db, loader),
		NewGetModelsCommand(cfg, db, loader),
		NewSetConversationContextCommand(db),
		NewResetConversationCommand(db),
	}
}

func buildHelpFromOperators(operators []string) string {
	operatorHelp := make([]string, len(operators))
	for i, o := range operators {
		operatorHelp[i] = CommandPrefix + o
	}

	return strings.Join(operatorHelp, "|")
}

func extractCommandValue(rawMsg string, operators []string) string {
	for _, op := range operators {
		r := regexp.MustCompile(fmt.Sprintf(`^@%s($|\s.*)`, op))
		foundResults := r.FindStringSubmatch(rawMsg)
		if len(foundResults) == 0 {
			continue
		}

		return strings.TrimSpace(foundResults[1])
	}

	return ""
}
