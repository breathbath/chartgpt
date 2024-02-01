package auth

import (
	"context"
	"time"

	"breathbathChatGPT/pkg/msg"

	logging "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type LoginHandler struct {
	us  *UserStorage
	cfg *Config
}

func NewLoginHandler(us *UserStorage, cfg *Config) *LoginHandler {
	return &LoginHandler{
		us:  us,
		cfg: cfg,
	}
}

func (lh *LoginHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	user := GetUserFromReq(req)

	if user == nil {
		return &msg.Response{
			Message: "your user is unknown",
			Type:    msg.Error,
		}, nil
	}

	return lh.handleNotVerifiedUser(ctx, req, user)
}

func (lh *LoginHandler) handleNotVerifiedUser(
	ctx context.Context,
	req *msg.Request,
	user *CachedUser,
) (*msg.Response, error) {
	log := logging.WithContext(ctx)

	log.Debugf("checking password for user %q", req.Sender.GetID())
	if !lh.checkPassword(req.Message, user) {
		log.Debugf("password %q for user %q is not correct", req.Message, req.Sender.GetID())
		return &msg.Response{
			Message: "please provide a valid password to access the bot functions",
			Type:    msg.Error,
		}, nil
	}
	log.Debugf("password for user %q is correct", req.Sender.GetID())

	user.State = UserVerified
	if lh.cfg.SessionDuration > 0 {
		user.LoginTill = time.Now().Add(lh.cfg.SessionDuration).Unix()
	}

	err := lh.us.WriteUserToStorage(ctx, user)
	if err != nil {
		return nil, err
	}

	op := &msg.Options{}
	op.WithIsResponseToHiddenMessage()

	return &msg.Response{
		Message: "Password is correct, you can continue using bot. Will delete the message with password for security reasons.",
		Type:    msg.Success,
		Options: op,
	}, nil
}

func (lh *LoginHandler) checkPassword(candidatePassword string, user *CachedUser) bool {
	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(candidatePassword))

	return err == nil
}

func (lh *LoginHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if lh.cfg.AuthIsDisabled {
		return false, nil
	}

	user := GetUserFromReq(req)

	if user != nil && user.State == UserVerified && (user.LoginTill == int64(0) || user.LoginTill > time.Now().Unix()) {
		return false, nil
	}

	return true, nil
}
