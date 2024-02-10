package auth

import (
	"context"
	"fmt"

	"breathbathChatGPT/pkg/help"
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/utils"

	"github.com/sirupsen/logrus"
)

type LogoutHandler struct {
	us      *UserStorage
	command string
}

func NewLogoutHandler(us *UserStorage) *LogoutHandler {
	return &LogoutHandler{
		us:      us,
		command: "/logout",
	}
}

func (h *LogoutHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	return utils.MatchesCommand(req.Message, h.command), nil
}

func (h *LogoutHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	user := GetUserFromReq(req)

	if user == nil {
		log.Warnf("user not found, will do nothing")
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: "User not found",
					Type:    msg.Success,
				},
			},
		}, nil
	}

	user.State = UserUnverified

	err := h.us.WriteUserToStorage(ctx, user)
	if err != nil {
		return nil, err
	}

	return &msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: "Logout success",
				Type:    msg.Success,
			},
		},
	}, nil
}

func (h *LogoutHandler) GetHelp(context.Context, *msg.Request) help.Result {
	text := fmt.Sprintf("%s: to logout from the system", h.command)

	return help.Result{Text: text}
}
