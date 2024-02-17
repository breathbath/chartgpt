package auth

import (
	"context"
	"fmt"
	"strings"

	"breathbathChatGPT/pkg/help"

	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/utils"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const (
	usersPrefix  = "users"
	usersVersion = "v1"
)

func GetAdminUserFromReq(req *msg.Request) *CachedUser {
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

func (us *UserStorage) ReadUserFromStorage(ctx context.Context, platform, userID string) (user *CachedUser, err error) {
	log := logrus.WithContext(ctx)

	cacheKey := us.generateUserCacheKey(platform, userID)
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

func (us *UserStorage) ReadUsersFromStorage(ctx context.Context, platform string) (users []CachedUser, err error) {
	parts := []string{
		usersVersion,
		platform,
		usersPrefix,
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
	return storage.GenerateCacheKey(usersVersion, platform, usersPrefix, login)
}

type UserMiddleware struct {
	us *UserStorage
}

func NewUserMiddleware(us *UserStorage) *UserMiddleware {
	return &UserMiddleware{us: us}
}

func (um UserMiddleware) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	platform := req.Platform
	userID := req.Sender.GetID()

	if platform == "" {
		return nil, errors.New("unknown message platform")
	}

	if userID == "" {
		return nil, errors.New("unknown message sender id")
	}

	u, err := um.us.ReadUserFromStorage(ctx, platform, userID)
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
	command       string
	us            *UserStorage
	adminDetector func(req *msg.Request) bool
}

func NewAddUserCommand(
	us *UserStorage,
	adminDetector func(req *msg.Request) bool,
) *AddUserCommand {
	return &AddUserCommand{
		command:       "/adduser",
		us:            us,
		adminDetector: adminDetector,
	}
}

func (au *AddUserCommand) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommand(req.Message, au.command) {
		return false, nil
	}

	if !au.adminDetector(req) {
		return false, nil
	}

	return true, nil
}

func (au *AddUserCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("got add user request")

	words := strings.Split(req.Message, " ")

	const expectedWordsCount = 4
	if len(words) < expectedWordsCount {
		log.Warnf("invalid user data provided: %q", req.Message)
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: "Invalid values provided, you need to provide all user details: login, platform and password as space separated values",
					Type:    msg.Error,
				},
			},
		}, nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(words[3]), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	u := &CachedUser{
		UID:          uuid.NewString(),
		Login:        strings.TrimPrefix(words[1], "@"),
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

	cachedUser, err := au.us.ReadUserFromStorage(ctx, u.PlatformName, u.Login)
	if err != nil {
		return nil, err
	}
	if cachedUser != nil && cachedUser.Role == AdminRole {
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: "Cannot downgrade user role from admin to user",
					Type:    msg.Error,
				},
			},
		}, nil
	}

	err = au.us.WriteUserToStorage(ctx, u)
	if err != nil {
		return nil, err
	}

	log.Debug("successfully added the user")

	op := &msg.Options{}
	op.WithIsResponseToHiddenMessage()

	return &msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: "successfully added user",
				Type:    msg.Success,
				Options: op,
			},
		},
	}, nil
}

func (au *AddUserCommand) GetHelp(_ context.Context, req *msg.Request) help.Result {
	user := GetAdminUserFromReq(req)

	if user == nil || user.Role != AdminRole {
		return help.Result{}
	}

	text := fmt.Sprintf(`%s #login# #platform# #password#: adds a new user, 
if success the initial message will be deleted for security reasons
to add a telegram user use your telegram user name without the at sign as #login# and 'telegram' as #platform#`, au.command)

	return help.Result{Text: text}
}

type ListUsersCommand struct {
	command       string
	us            *UserStorage
	adminDetector func(req *msg.Request) bool
}

func NewListUsersCommand(
	us *UserStorage,
	adminDetector func(req *msg.Request) bool,
) *ListUsersCommand {
	return &ListUsersCommand{
		command:       "/users",
		us:            us,
		adminDetector: adminDetector,
	}
}

func (lu *ListUsersCommand) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !strings.HasPrefix(req.Message, lu.command) {
		return false, nil
	}

	if !lu.adminDetector(req) {
		return false, nil
	}

	return true, nil
}

func (lu *ListUsersCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("got list users request")

	users, err := lu.us.ReadUsersFromStorage(ctx, "telegram")
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
		Messages: []msg.ResponseMessage{
			{
				Message: strings.Join(usersRaw, "\n"),
				Type:    msg.Success,
			},
		},
	}, nil
}

func (lu *ListUsersCommand) GetHelp(_ context.Context, req *msg.Request) help.Result {
	user := GetAdminUserFromReq(req)

	if user == nil || user.Role != AdminRole {
		return help.Result{}
	}

	text := fmt.Sprintf(`%s: lists current users`, lu.command)

	return help.Result{Text: text}
}

type DeleteUserCommand struct {
	command       string
	us            *UserStorage
	adminDetector func(req *msg.Request) bool
}

func NewDeleteUserCommand(
	us *UserStorage,
	adminDetector func(req *msg.Request) bool,
) *DeleteUserCommand {
	return &DeleteUserCommand{
		command:       "/deluser",
		us:            us,
		adminDetector: adminDetector,
	}
}

func (du *DeleteUserCommand) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !strings.HasPrefix(req.Message, du.command) {
		return false, nil
	}

	if !du.adminDetector(req) {
		return false, nil
	}

	return true, nil
}

func (du *DeleteUserCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("got delete user request")

	inputID := utils.ExtractCommandValue(req.Message, du.command)
	if inputID == "" {
		log.Warnf("no user id provided in %q", req.Message)
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: "no user id provided",
					Type:    msg.Error,
				},
			},
		}, nil
	}

	users, err := du.us.ReadUsersFromStorage(ctx, "telegram")
	if err != nil {
		return nil, err
	}

	for i, u := range users {
		if u.UID != inputID && u.Login != inputID {
			continue
		}

		log.Debugf("found user %s by id %s, will delete it", u.String(), inputID)

		err := du.us.DeleteUser(ctx, &users[i])
		if err != nil {
			return nil, err
		}

		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: fmt.Sprintf("successfully deleted user %q", inputID),
					Type:    msg.Success,
				},
			},
		}, nil
	}

	return &msg.Response{
		Messages: []msg.ResponseMessage{
			{
				Message: fmt.Sprintf("didn't find user by %q", inputID),
				Type:    msg.Error,
			},
		},
	}, nil
}

func (du *DeleteUserCommand) GetHelp(_ context.Context, req *msg.Request) help.Result {
	user := GetAdminUserFromReq(req)

	if user == nil || user.Role != AdminRole {
		return help.Result{}
	}

	text := fmt.Sprintf(`%s #user id or login#: deletes the requested user`, du.command)

	return help.Result{Text: text}
}
