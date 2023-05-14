package cmd

import (
	"breathbathChatGPT/pkg/auth"
	"breathbathChatGPT/pkg/chatgpt"
	"breathbathChatGPT/pkg/help"
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/telegram"
)

func BuildMessageHandler(db storage.Client) (msg.Handler, error) {
	handlerComposite := msg.HandlerComposite{
		Handlers: []msg.Handler{},
	}

	loginHandler, err := auth.BuildLoginHandler(db)
	if err != nil {
		return nil, err
	}

	logoutHandler := auth.NewLogoutHandler(db)

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

	helpHandler := &help.Handler{
		Providers: []help.Provider{
			logoutHandler,
			setModelHandler,
			setConversationCtxHandler,
			getModelsHandler,
			resetConversationHandler,
		},
	}

	handlerComposite.Handlers = append(
		handlerComposite.Handlers,
		startHandler,
		loginHandler,
		helpHandler,
		logoutHandler,
		setConversationCtxHandler,
		resetConversationHandler,
		setModelHandler,
		getModelsHandler,
		chatCompletionHandler,
	)

	return handlerComposite, nil
}
