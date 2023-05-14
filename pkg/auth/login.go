package auth

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"context"
	"github.com/pkg/errors"
	logging "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type LoginHandler struct {
	db  storage.Client
	cfg *Config
}

func NewLoginHandler(db storage.Client, cfg *Config) *LoginHandler {
	return &LoginHandler{
		db:  db,
		cfg: cfg,
	}
}

func (h *LoginHandler) writeUserToStorage(ctx context.Context, u *CachedUser) error {
	ctxValue := context.WithValue(ctx, storage.IsNotLoggableContentCtxKey, true)

	cacheKey := GenerateUserCacheKey(u.PlatformName, u.Id)

	return h.db.Save(ctxValue, cacheKey, u, 0)
}

func (h *LoginHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	user, err := h.findUserInCache(ctx, req)
	if err != nil {
		return nil, err
	}

	if user == nil {
		return &msg.Response{
			Message: "your user is unknown",
			Type:    msg.Error,
		}, nil
	}

	return h.handleNotVerifiedUser(ctx, req, user)
}

func (h *LoginHandler) findUserInCache(ctx context.Context, req *msg.Request) (u *CachedUser, err error) {
	platform := req.Platform
	userId := req.Sender.GetID()

	cacheKey := GenerateUserCacheKey(platform, userId)
	u = new(CachedUser)
	var isFound bool
	isFound, err = h.db.Load(ctx, cacheKey, u)
	if err != nil {
		return nil, err
	}

	if !isFound {
		return nil, nil
	}

	return u, nil
}

func (h *LoginHandler) handleNotVerifiedUser(
	ctx context.Context,
	req *msg.Request,
	user *CachedUser,
) (*msg.Response, error) {
	log := logging.WithContext(ctx)

	log.Debugf("checking password for user %q", req.Sender.GetID())
	if !h.checkPassword(req.Message, user) {
		log.Debugf("password %q for user %q is not correct", req.Message, req.Sender.GetID())
		return &msg.Response{
			Message: "please provide a valid password to access the bot functions",
			Type:    msg.Error,
		}, nil
	}
	log.Debugf("password for user %q is correct", req.Sender.GetID())

	user.State = UserVerified
	if h.cfg.SessionDuration > 0 {
		user.LoginTill = time.Now().Add(h.cfg.SessionDuration).Unix()
	}

	err := h.writeUserToStorage(ctx, user)
	if err != nil {
		return nil, err
	}

	return &msg.Response{
		Message: "Password is correct, you can continue using bot. Will delete the message with password for security reasons.",
		Type:    msg.Success,
		Meta: map[string]interface{}{
			"is_hidden_message": true,
		},
	}, nil
}

func (a *LoginHandler) checkPassword(candidatePassword string, user *CachedUser) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(candidatePassword))

	return err == nil
}

func (a *LoginHandler) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	if req.Platform == "" {
		return false, errors.New("unknown message platform")
	}

	if req.Sender.GetID() == "" {
		return false, errors.New("unknown message sender id")
	}

	user, err := a.findUserInCache(ctx, req)
	if err != nil {
		return false, err
	}

	if user != nil && user.State == UserVerified && (user.LoginTill == int64(0) || user.LoginTill > time.Now().Unix()) {
		return false, nil
	}

	return true, nil
}
