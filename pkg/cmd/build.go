package cmd

import (
	"breathbathChatGPT/pkg/auth"
	"breathbathChatGPT/pkg/chatgpt"
	"breathbathChatGPT/pkg/help"
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/recommend"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/telegram"
	"gorm.io/gorm"
)

func BuildMessageRouter(
	cacheClient storage.Client,
	dbConn *gorm.DB,
) (*msg.Router, error) {
	us := auth.NewUserStorage(cacheClient)

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
		usr := auth.GetAdminUserFromReq(req)
		return usr != nil && usr.Role == auth.AdminRole
	}

	setConversationCtxHandler := chatgpt.NewSetConversationContextCommand(cacheClient, isScopedModeFunc, isAdminDetector)
	resetConversationHandler := chatgpt.NewResetConversationHandler(cacheClient, isScopedModeFunc, isAdminDetector)

	validationErr := chartGptCfg.Validate()
	if validationErr.HasErrors() {
		return nil, validationErr
	}

	loader := chatgpt.NewSettingsLoader(cacheClient, chartGptCfg, isScopedModeFunc)

	setModelHandler := chatgpt.NewSetModelHandler(chartGptCfg, cacheClient, loader, isScopedModeFunc, isAdminDetector)

	getModelsHandler := chatgpt.NewGetModelsCommand(chartGptCfg, cacheClient, loader, isScopedModeFunc, isAdminDetector)

	wineProvider := recommend.NewWineProvider(dbConn)

	dialogHandler := recommend.NewDialogHandler(dbConn)

	chatCompletionHandler, err := chatgpt.NewChatCompletionHandler(
		chartGptCfg,
		cacheClient,
		loader,
		isScopedModeFunc,
		wineProvider,
		dbConn,
		dialogHandler,
	)
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
	likeHandler := recommend.NewLikeHandler(dbConn, chatCompletionHandler)
	addToFavoritesHandler := recommend.NewAddToFavoritesHandler(dbConn, chatCompletionHandler)
	listFavorites := recommend.NewListFavoritesHandler(dbConn, chatCompletionHandler)
	delFromFavorites := recommend.NewDeleteFromFavoritesHandler(dbConn, chatCompletionHandler)

	r := &msg.Router{
		Handlers: []msg.Handler{
			startHandler,
			loginHandler,
			helpHandler,
			logoutHandler,
			likeHandler,
			addToFavoritesHandler,
			listFavorites,
			delFromFavorites,
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
