package auth

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strings"
)

type LogoutHandler struct {
	db      storage.Client
	command string
}

func NewLogoutHandler(db storage.Client) *LogoutHandler {
	return &LogoutHandler{
		db:      db,
		command: "/logout",
	}
}

func (h *LogoutHandler) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, h.command), nil
}

func (h *LogoutHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	if req.Sender.GetID() == "" {
		return nil, errors.New("unknown message sender id")
	}

	platform := req.Platform
	userId := req.Sender.GetID()

	cacheKey := GenerateUserCacheKey(platform, userId)

	u := new(CachedUser)
	var isFound bool
	var err error
	isFound, err = h.db.Load(ctx, cacheKey, u)
	if err != nil {
		return nil, err
	}
	if !isFound {
		log.Warnf("user not found by %q, will do nothing", cacheKey)
		return &msg.Response{
			Message: "User not found",
			Type:    msg.Success,
		}, nil
	}

	u.State = UserUnverified
	err = h.db.Save(ctx, cacheKey, u, 0)
	if err != nil {
		return nil, err
	}

	return &msg.Response{
		Message: "Logout success",
		Type:    msg.Success,
	}, nil
}

func (h *LogoutHandler) GetHelp() string {
	return fmt.Sprintf("%s: to logout from the system", h.command)
}
