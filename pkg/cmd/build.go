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

	setConversationCtxHandler := chatgpt.NewSetConversationContextCommand(db)
	resetConversationHandler := chatgpt.NewResetConversationHandler(db)

	chartGptCfg, err := chatgpt.LoadConfig()
	if err != nil {
		return nil, err
	}

	validationErr := chartGptCfg.Validate()
	if validationErr.HasErrors() {
		return nil, validationErr
	}

	loader := chatgpt.NewSettingsLoader(db, chartGptCfg)

	setModelHandler := chatgpt.NewSetModelHandler(chartGptCfg, db, loader)

	getModelsHandler := chatgpt.NewGetModelsCommand(chartGptCfg, db, loader)

	chatCompletionHandler, err := chatgpt.NewChatCompletionHandler(chartGptCfg, db, loader)
	if err != nil {
		return nil, err
	}

	addUserHandler := auth.NewAddUserCommand(us)
	listUsersHandler := auth.NewListUsersCommand(us)
	deleteUsersHandler := auth.NewDeleteUserCommand(us)

	helpHandler := &help.Handler{
		Providers: []help.Provider{
			setModelHandler,
			setConversationCtxHandler,
			getModelsHandler,
			resetConversationHandler,
			addUserHandler,
			listUsersHandler,
			deleteUsersHandler,
			logoutHandler,
		},
	}

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
