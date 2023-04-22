package auth

import (
	"breathbathChartGPT/pkg/msg"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	logging "github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"strings"
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
		for _, id := range u.UserIDs {
			if strings.ToLower(id) == strings.ToLower(userId) {
				return &u
			}
		}
	}

	return nil
}

func (a *Handler) buildUserStorageKey(userId, platformId string) string {
	return fmt.Sprintf("auth_handler_%s_user_id_%s", strings.ToLower(platformId), strings.ToLower(userId))
}

func (a *Handler) findUserInStorage(ctx context.Context, userId, platformId string) (*User, error) {
	userBytes, found, err := a.storage.Read(ctx, a.buildUserStorageKey(userId, platformId))
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

func (a *Handler) writeUserToStorage(ctx context.Context, userId, platform string, u *User) error {
	rawBytes, err := json.Marshal(u)
	if err != nil {
		return errors.Wrapf(err, "failed to convert user to json")
	}

	return a.storage.Write(ctx, a.buildUserStorageKey(userId, platform), rawBytes, a.cfg.SessionDuration)
}

func (a *Handler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logging.WithContext(ctx)

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

	userFromStorage, err := a.findUserInStorage(ctx, req.Sender.GetID(), req.Source)
	if err != nil {
		return nil, err
	}

	if req.Message == "/auth" {
		log.Infof("auth command received, expecting password from prompt")
		userFromConfig.State = UserReadyToBeVerified
		err = a.writeUserToStorage(ctx, req.Sender.GetID(), req.Source, userFromConfig)
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
		log.Infof("user is not authenticated and didn't receive auth command, waiting for a valid password")
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
		log.Infof("checking password for user %q", req.Sender.GetID())

		if a.checkPassword(req.Message, userFromConfig) {
			log.Infof("password for user %q is correct", req.Sender.GetID())
			userFromConfig.State = UserVerified
			err = a.writeUserToStorage(ctx, req.Sender.GetID(), req.Source, userFromConfig)
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
		log.Infof("password %q for user %q is incorrect, expecting another password", req.Message, req.Sender.GetID())

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
