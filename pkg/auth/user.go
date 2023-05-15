package auth

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/utils"
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"strings"
)

const usersPrefix = "users"
const usersVersion = "v1"

func GetUserFromReq(req *msg.Request) *CachedUser {
	userI, ok := req.Meta["curUser"]
	if !ok {
		return nil
	}

	return userI.(*CachedUser)
}

type UserStorage struct {
	db storage.Client
}

func NewUserStorage(db storage.Client) *UserStorage {
	return &UserStorage{db: db}
}

func (us *UserStorage) WriteUserToStorage(ctx context.Context, u *CachedUser) error {
	log := logrus.WithContext(ctx)
	ctxValue := context.WithValue(ctx, storage.IsNotLoggableContentCtxKey, true)

	cacheKey := us.generateUserCacheKey(u.PlatformName, u.Login)

	err := us.db.Save(ctxValue, cacheKey, u, 0)
	if err != nil {
		return err
	}

	log.Debugf("wrote user %s under %q", u.String(), cacheKey)

	return nil
}

func (us *UserStorage) ReadUserFromStorage(ctx context.Context, platform, userId string) (user *CachedUser, err error) {
	log := logrus.WithContext(ctx)

	cacheKey := us.generateUserCacheKey(platform, userId)
	u := new(CachedUser)

	ctxValue := context.WithValue(ctx, storage.IsNotLoggableContentCtxKey, true)
	isFound, err := us.db.Load(ctxValue, cacheKey, u)
	if err != nil {
		return nil, err
	}

	if !isFound {
		return nil, nil
	}

	log.Debugf("successfully read user %s under %q", u.String(), cacheKey)

	return u, nil
}

func (us *UserStorage) ReadUsersFromStorage(ctx context.Context) (users []CachedUser, err error) {
	parts := []string{
		usersPrefix,
		usersVersion,
		"*",
	}

	keys, err := us.db.FindKeys(ctx, strings.Join(parts, "/"))
	if err != nil {
		return nil, err
	}

	users = make([]CachedUser, 0, len(keys))
	ctxValue := context.WithValue(ctx, storage.IsNotLoggableContentCtxKey, true)

	for _, key := range keys {
		u := new(CachedUser)

		_, err := us.db.Load(ctxValue, key, u)
		if err != nil {
			return nil, err
		}

		users = append(users, *u)
	}

	return users, nil
}

func (us *UserStorage) DeleteUser(ctx context.Context, u *CachedUser) error {
	log := logrus.WithContext(ctx)

	cacheKey := us.generateUserCacheKey(u.PlatformName, u.Login)

	err := us.db.Delete(ctx, cacheKey)
	if err != nil {
		return err
	}

	log.Debugf("deleted a user under key %q", cacheKey)

	return nil
}

func (us *UserStorage) generateUserCacheKey(platform, login string) string {
	parts := []string{
		usersPrefix,
		usersVersion,
		strings.ToLower(platform),
		strings.ToLower(login),
	}

	return strings.Join(parts, "/")
}

type UserMiddleware struct {
	us *UserStorage
}

func NewUserMiddleware(us *UserStorage) *UserMiddleware {
	return &UserMiddleware{us: us}
}

func (um UserMiddleware) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	platform := req.Platform
	userId := req.Sender.GetID()

	if platform == "" {
		return nil, errors.New("unknown message platform")
	}

	if userId == "" {
		return nil, errors.New("unknown message sender id")
	}

	u, err := um.us.ReadUserFromStorage(ctx, platform, userId)
	if err != nil {
		return nil, err
	}

	if u == nil {
		return nil, nil
	}

	req.Meta["curUser"] = u

	return nil, nil
}

type AddUserCommand struct {
	command string
	us      *UserStorage
}

func NewAddUserCommand(us *UserStorage) *AddUserCommand {
	return &AddUserCommand{
		command: "/adduser",
		us:      us,
	}
}

func CheckAdmin(ctx context.Context, req *msg.Request) bool {
	log := logrus.WithContext(ctx)

	user := GetUserFromReq(req)

	if user == nil || user.Role != AdminRole {
		log.Warnf("unauthorised attempt to access admin action, provided user data: %s", user.String())
		return false
	}

	return true
}

func (au *AddUserCommand) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, au.command), nil
}

func (au *AddUserCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("got add user request")

	if !CheckAdmin(ctx, req) {
		return nil, errors.New("you need to be admin to add a user")
	}

	words := strings.Split(req.Message, " ")

	if len(words) < 4 {
		log.Warnf("invalid user data provided: %q", req.Message)
		return &msg.Response{
			Message: "Invalid values provided, you need to provide all user details: login, platform and password as space separated values",
			Type:    msg.Error,
		}, nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(words[3]), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	u := &CachedUser{
		Uid:          uuid.NewString(),
		Login:        words[1],
		State:        UserUnverified,
		PlatformName: words[2],
		Role:         UserRole,
		PasswordHash: string(hashedPassword),
	}

	log.Debugf(
		"will add user (id: %s, state: %d, platform: %s, role: %s, password length: %d)",
		u.Login,
		u.State,
		u.PlatformName,
		u.Role,
		len(words[3]),
	)

	err = au.us.WriteUserToStorage(ctx, u)
	if err != nil {
		return nil, err
	}

	log.Debug("successfully added the user")

	return &msg.Response{
		Message: "successfully added user",
		Type:    msg.Success,
		Meta: map[string]interface{}{
			"is_hidden_message": true,
		},
	}, nil
}

func (au *AddUserCommand) GetHelp(ctx context.Context, req *msg.Request) string {
	user := GetUserFromReq(req)

	if user == nil || user.Role != AdminRole {
		return ""
	}

	return fmt.Sprintf(`%s #login# #platform# #password#: adds a new user, 
if success the initial message will be deleted for security reasons
to add a telegram user use your telegram user name without the at sign as #login# and 'telegram' as #platform#`, au.command)
}

type ListUsersCommand struct {
	command string
	us      *UserStorage
}

func NewListUsersCommand(us *UserStorage) *ListUsersCommand {
	return &ListUsersCommand{
		command: "/users",
		us:      us,
	}
}

func (lu *ListUsersCommand) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, lu.command), nil
}

func (lu *ListUsersCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("got list users request")

	if !CheckAdmin(ctx, req) {
		return nil, errors.New("you need to be admin to add a user")
	}

	users, err := lu.us.ReadUsersFromStorage(ctx)
	if err != nil {
		return nil, err
	}

	log.Debugf(
		"got %d users from storage",
		len(users),
	)

	usersRaw := make([]string, len(users))

	for i, u := range users {
		usersRaw[i] = " " + u.String()
	}

	return &msg.Response{
		Message: strings.Join(usersRaw, "\n"),
		Type:    msg.Success,
	}, nil
}

func (lu *ListUsersCommand) GetHelp(ctx context.Context, req *msg.Request) string {
	user := GetUserFromReq(req)

	if user == nil || user.Role != AdminRole {
		return ""
	}

	return fmt.Sprintf(`%s: lists current users`, lu.command)
}

type DeleteUserCommand struct {
	command string
	us      *UserStorage
}

func NewDeleteUserCommand(us *UserStorage) *DeleteUserCommand {
	return &DeleteUserCommand{
		command: "/deluser",
		us:      us,
	}
}

func (du *DeleteUserCommand) CanHandle(ctx context.Context, req *msg.Request) (bool, error) {
	return strings.HasPrefix(req.Message, du.command), nil
}

func (du *DeleteUserCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("got delete user request")

	if !CheckAdmin(ctx, req) {
		return nil, errors.New("you need to be admin to delete a user")
	}

	inputId := utils.ExtractCommandValue(req.Message, du.command)
	if inputId == "" {
		log.Warnf("no user id provided in %q", req.Message)
		return &msg.Response{
			Message: "no user id provided",
			Type:    msg.Error,
		}, nil
	}

	users, err := du.us.ReadUsersFromStorage(ctx)
	if err != nil {
		return nil, err
	}

	for _, u := range users {
		if u.Uid != inputId && u.Login != inputId {
			continue
		}

		log.Debugf("found user %s by id %s, will delete it", u.String(), inputId)
		err := du.us.DeleteUser(ctx, &u)
		if err != nil {
			return nil, err
		}

		return &msg.Response{
			Message: fmt.Sprintf("successfully deleted user %q", inputId),
			Type:    msg.Success,
		}, nil
	}

	return &msg.Response{
		Message: fmt.Sprintf("didn't find user by %q", inputId),
		Type:    msg.Error,
	}, nil
}

func (du *DeleteUserCommand) GetHelp(ctx context.Context, req *msg.Request) string {
	user := GetUserFromReq(req)

	if user == nil || user.Role != AdminRole {
		return ""
	}

	return fmt.Sprintf(`%s #user id or login#: deletes the requested user`, du.command)
}
