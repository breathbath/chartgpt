package auth

import (
	"breathbathChartGPT/pkg/msg"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type Storage interface {
	Read(ctx context.Context, userId string) (raw []byte, found bool, err error)
	Write(ctx context.Context, userId string, raw []byte, exp time.Duration) error
}

type Handler struct {
	passHandler msg.Handler
	storage     Storage
	cfg         *Config
}

func NewHandler(passHandler msg.Handler, storage Storage, cfg *Config) (*Handler, error) {
	err := cfg.Validate()
	if err.HasErrors() {
		return nil, err
	}

	return &Handler{
		passHandler: passHandler,
		storage:     storage,
		cfg:         cfg,
	}, nil
}

func (a *Handler) findUserInConfig(userId string) *User {
	for _, u := range a.cfg.Users {
		if u.UserID == userId {
			return &u
		}
	}

	return nil
}

func (a *Handler) buildUserStorageKey(userId string) string {
	return "auth_handler_user_id_" + userId
}

func (a *Handler) findUserInStorage(ctx context.Context, userId string) (*User, error) {
	userBytes, found, err := a.storage.Read(ctx, a.buildUserStorageKey(userId))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read data from storage")
	}

	if !found {
		return nil, nil
	}

	u := new(User)
	err = json.Unmarshal(userBytes, u)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert %q to user model", string(userBytes))
	}

	return u, nil
}

func (a *Handler) writeUserToStorage(ctx context.Context, u *User) error {
	rawBytes, err := json.Marshal(u)
	if err != nil {
		return errors.Wrapf(err, "failed to convert user to json")
	}

	return a.storage.Write(ctx, a.buildUserStorageKey(u.UserID), rawBytes, a.cfg.SessionDuration)
}

func (a *Handler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	if req.Message == "" {
		return nil, nil
	}

	if req.Sender.GetID() == "" {
		return nil, errors.New("unknown message sender id")
	}

	userFromConfig := a.findUserInConfig(req.Sender.GetID())
	if userFromConfig == nil {
		return nil, errors.Errorf("user with the provided id %q is not configured", req.Sender.GetID())
	}

	userFromStorage, err := a.findUserInStorage(ctx, req.Sender.GetID())
	if err != nil {
		return nil, err
	}

	if req.Message == "/auth" {
		userFromConfig.State = UserReadyToBeVerified
		err = a.writeUserToStorage(ctx, userFromConfig)
		if err != nil {
			return nil, err
		}
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: "Please provide your password",
					Type:    msg.Success,
				},
			},
		}, nil
	}

	if userFromStorage == nil || userFromStorage.State == UserUnverified {
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: "Authenticate",
					Type:    msg.Prompt,
					Meta: map[string]interface{}{
						"data": "/auth",
					},
				},
			},
		}, nil
	}

	if userFromStorage.State == UserReadyToBeVerified {
		if a.checkPassword(req.Message, userFromConfig) {
			userFromConfig.State = UserVerified
			err = a.writeUserToStorage(ctx, userFromConfig)
			if err != nil {
				return nil, err
			}
			return &msg.Response{
				Messages: []msg.ResponseMessage{
					{
						Message: "Password is correct, you can continue using bot. You can delete the message with password for security reasons.",
						Type:    msg.Success,
					},
				},
			}, nil
		}
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: "Password is incorrect, please provide a valid password",
					Type:    msg.Error,
				},
			},
		}, nil
	}

	if userFromStorage.State == UserVerified {
		return a.passHandler.Handle(ctx, req)
	}

	return nil, errors.Errorf("unsupported user state: %d", userFromStorage.State)
}

func (a *Handler) checkPassword(candidatePassword string, userFromConfig *User) bool {
	err := bcrypt.CompareHashAndPassword([]byte(userFromConfig.PasswordHash), []byte(candidatePassword))

	return err == nil
}
