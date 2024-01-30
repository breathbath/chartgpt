package cmd

import (
	"breathbathChatGPT/pkg/auth"
	"breathbathChatGPT/pkg/chatgpt"
	"breathbathChatGPT/pkg/help"
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/telegram"
)

func BuildMessageRouter(db storage.Client) (*msg.Router, error) {
	us := auth.NewUserStorage(db)

	userMiddleware := auth.NewUserMiddleware(us)

	loginHandler, err := auth.BuildLoginHandler(us)
	if err != nil {
		return nil, err
	}

	logoutHandler := auth.NewLogoutHandler(us)

	startHandler := &telegram.StartHandler{}

	chartGptCfg, err := chatgpt.LoadConfig()
	if err != nil {
		return nil, err
	}

	isScopedModeFunc := func() bool {
		return chartGptCfg.ScopedMode
	}

	isAdminDetector := func(req *msg.Request) bool {
		usr := auth.GetUserFromReq(req)
		return usr != nil && usr.Role == auth.AdminRole
	}

	setConversationCtxHandler := chatgpt.NewSetConversationContextCommand(db, isScopedModeFunc, isAdminDetector)
	resetConversationHandler := chatgpt.NewResetConversationHandler(db, isScopedModeFunc, isAdminDetector)

	validationErr := chartGptCfg.Validate()
	if validationErr.HasErrors() {
		return nil, validationErr
	}

	loader := chatgpt.NewSettingsLoader(db, chartGptCfg, isScopedModeFunc)

	setModelHandler := chatgpt.NewSetModelHandler(chartGptCfg, db, loader, isScopedModeFunc, isAdminDetector)

	getModelsHandler := chatgpt.NewGetModelsCommand(chartGptCfg, db, loader, isScopedModeFunc, isAdminDetector)

	chatCompletionHandler, err := chatgpt.NewChatCompletionHandler(chartGptCfg, db, loader, isScopedModeFunc)
	if err != nil {
		return nil, err
	}

	addUserHandler := auth.NewAddUserCommand(us, isAdminDetector)
	listUsersHandler := auth.NewListUsersCommand(us, isAdminDetector)
	deleteUsersHandler := auth.NewDeleteUserCommand(us, isAdminDetector)

	helpProviders := []help.Provider{
		setModelHandler,
		setConversationCtxHandler,
		getModelsHandler,
		resetConversationHandler,
		addUserHandler,
		listUsersHandler,
		deleteUsersHandler,
		logoutHandler,
	}
	helpHandler := help.NewHandler(isScopedModeFunc, isAdminDetector, helpProviders)

	r := &msg.Router{
		Handlers: []msg.Handler{
			startHandler,
			loginHandler,
			helpHandler,
			logoutHandler,
			setConversationCtxHandler,
			resetConversationHandler,
			setModelHandler,
			getModelsHandler,
			addUserHandler,
			listUsersHandler,
			deleteUsersHandler,
			chatCompletionHandler,
		},
	}

	r.UseMiddleware(userMiddleware)

	return r, nil
}
