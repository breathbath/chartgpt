package auth

import (
	"context"
	"fmt"
	"strings"

	"breathbathChatGPT/pkg/msg"

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

func (h *LogoutHandler) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, h.command), nil
}

func (h *LogoutHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	user := GetUserFromReq(req)

	if user == nil {
		log.Warnf("user not found, will do nothing")
		return &msg.Response{
			Message: "User not found",
			Type:    msg.Success,
		}, nil
	}

	user.State = UserUnverified

	err := h.us.WriteUserToStorage(ctx, user)
	if err != nil {
		return nil, err
	}

	return &msg.Response{
		Message: "Logout success",
		Type:    msg.Success,
	}, nil
}

func (h *LogoutHandler) GetHelp(ctx context.Context, req *msg.Request) string {
	return fmt.Sprintf("%s: to logout from the system", h.command)
}
