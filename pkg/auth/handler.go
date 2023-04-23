package auth

import (
	"breathbathChartGPT/pkg/errs"
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
	Read(ctx context.Context, key string) (raw []byte, found bool, err error)
	Write(ctx context.Context, key string, raw []byte, exp time.Duration) error
	Delete(ctx context.Context, key string) error
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

func (h *Handler) findUserInConfig(userId string) *ConfiguredUser {
	for _, u := range h.cfg.Users {
		for _, id := range u.UserIDs {
			if strings.ToLower(id) == strings.ToLower(userId) {
				return &u
			}
		}
	}

	return nil
}

func (h *Handler) buildUserStorageKey(req *msg.Request) string {
	conversationIdI, ok := req.Meta["conversation_id"]
	conversationId := ""
	if ok {
		conversationId = fmt.Sprint(conversationIdI)
	}
	return fmt.Sprintf("%s/%s/%s", strings.ToLower(req.Source), conversationId, strings.ToLower(req.Sender.GetID()))
}

func (h *Handler) findUserInStorage(ctx context.Context, cacheKey string) (*CachedUser, error) {
	userBytes, found, err := h.storage.Read(ctx, cacheKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read data from storage")
	}

	if !found {
		return nil, nil
	}

	u := new(CachedUser)
	err = json.Unmarshal(userBytes, u)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to convert %q to user model", string(userBytes))
	}

	return u, nil
}

func (h *Handler) writeUserToStorage(ctx context.Context, cacheKey string, u *CachedUser) error {
	rawBytes, err := json.Marshal(u)
	if err != nil {
		return errors.Wrapf(err, "failed to convert user to json")
	}

	return h.storage.Write(ctx, cacheKey, rawBytes, h.cfg.SessionDuration)
}

func (h *Handler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	if req.Message == "" {
		return nil, nil
	}

	if req.Sender.GetID() == "" {
		return nil, errors.New("unknown message sender id")
	}

	userFromConfig := h.findUserInConfig(req.Sender.GetID())
	if userFromConfig == nil {
		return &msg.Response{
			Message: "your user is not registered to work with this bot",
			Type:    msg.Error,
		}, nil
	}

	cacheKey := h.buildUserStorageKey(req)

	userFromStorage, err := h.findUserInStorage(ctx, cacheKey)
	if err != nil {
		return nil, err
	}

	if userFromStorage == nil || userFromStorage.State != UserVerified {
		return h.handleNotVerifiedUser(ctx, req, userFromConfig, cacheKey)
	}

	if msg.MatchCommand(req.Message, []string{"logout"}) {
		return h.handleLogout(ctx, cacheKey)
	}

	return h.passHandler.Handle(ctx, req)
}

func (h *Handler) handleLogout(ctx context.Context, cacheKey string) (*msg.Response, error) {
	log := logging.WithContext(ctx)
	log.Infof("got logout command, will logout current user")

	err := h.storage.Delete(ctx, cacheKey)

	if err != nil {
		errs.Handle(err, false)
		return &msg.Response{
			Message: "failed to logout",
			Type:    msg.Error,
		}, nil
	}

	return &msg.Response{
		Message: "logout success",
		Type:    msg.Success,
	}, nil
}

func (h *Handler) handleNotVerifiedUser(
	ctx context.Context,
	req *msg.Request,
	userFromConfig *ConfiguredUser,
	cacheKey string,
) (*msg.Response, error) {
	log := logging.WithContext(ctx)
	log.Infof("user %q is not authenticated as it's not found in the cache", req.Sender.GetID())

	log.Infof("checking password for user %q", req.Sender.GetID())
	if !h.checkPassword(req.Message, userFromConfig) {
		log.Infof("password %q for user %q is not correct", req.Message, req.Sender.GetID())
		return &msg.Response{
			Message: "please provide a valid password to access the bot functions",
			Type:    msg.Error,
		}, nil
	}
	log.Infof("password for user %q is correct", req.Sender.GetID())

	cachedUser := &CachedUser{
		Id:       req.Sender.GetID(),
		State:    UserVerified,
		Platform: req.Source,
	}

	err := h.writeUserToStorage(ctx, cacheKey, cachedUser)
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

func (a *Handler) checkPassword(candidatePassword string, userFromConfig *ConfiguredUser) bool {
	err := bcrypt.CompareHashAndPassword([]byte(userFromConfig.PasswordHash), []byte(candidatePassword))

	return err == nil
}
