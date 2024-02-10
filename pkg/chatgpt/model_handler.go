package chatgpt

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"breathbathChatGPT/pkg/help"

	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/rest"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/utils"

	"github.com/sirupsen/logrus"
)

type SetModelHandler struct {
	ModelCommand
	commands      []string
	loader        *Loader
	modeDetector  func() bool
	adminDetector func(req *msg.Request) bool
}

func NewSetModelHandler(
	cfg *Config,
	db storage.Client,
	loader *Loader,
	modeDetector func() bool,
	adminDetector func(req *msg.Request) bool,
) *SetModelHandler {
	return &SetModelHandler{
		ModelCommand: ModelCommand{
			cfg: cfg,
			db:  db,
		},
		commands:      []string{"/setmodel", "/model", "/savemodel"},
		loader:        loader,
		modeDetector:  modeDetector,
		adminDetector: adminDetector,
	}
}

type ModelCommand struct {
	cfg *Config
	db  storage.Client
}

func (mc *ModelCommand) getSupportedModelIDs(ctx context.Context) ([]string, error) {
	modelsResp := new(ModelsResponse)
	reqsr := rest.NewRequester(ModelsURL, modelsResp)
	reqsr.WithBearer(mc.cfg.APIKey)

	const defaultModelsCacheValidityHours = 24
	const modelsVersion = "v1"

	cacheKey := storage.GenerateCacheKey(modelsVersion, "chatgpt", "models", "rest")
	reqsr.WithCache(cacheKey, mc.db, time.Hour*defaultModelsCacheValidityHours)

	err := reqsr.Request(ctx)
	if err != nil {
		return nil, err
	}

	modelIDs := make([]string, len(modelsResp.Models))
	for i := range modelsResp.Models {
		modelIDs[i] = modelsResp.Models[i].ID
	}

	return modelIDs, nil
}

func (smc *SetModelHandler) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommands(req.Message, smc.commands) {
		return false, nil
	}

	if smc.modeDetector() && !smc.adminDetector(req) {
		return false, nil
	}

	return true, nil
}

func (smc *SetModelHandler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	modelName := ""
	for _, c := range smc.commands {
		modelName = utils.ExtractCommandValue(req.Message, c)
		if modelName != "" {
			break
		}
	}

	if modelName == "" {
		return &msg.Response{
			Message: "empty model name provided",
			Type:    msg.Error,
		}, nil
	}

	log.Debugf("got set model command: %q", modelName)

	isModelSupported, err := smc.isModelSupported(ctx, modelName)
	if err != nil {
		return nil, err
	}

	if !isModelSupported {
		log.Errorf("unsupported model name: %q", modelName)
		return &msg.Response{
			Message: fmt.Sprintf("unsupported model name %q", modelName),
			Type:    msg.Error,
		}, nil
	}

	err = smc.loader.SaveModel(ctx, &ConfiguredModel{Model: modelName}, req)
	if err != nil {
		return nil, err
	}

	log.Debugf("saved current model setting %q", modelName)

	return &msg.Response{
		Message: fmt.Sprintf("successfully set the current model for all requests to %q", modelName),
		Type:    msg.Success,
	}, nil
}

func (smc *SetModelHandler) GetHelp(context.Context, *msg.Request) help.Result {
	text := fmt.Sprintf("%s #modelName#: to change the active ChatGPT model", strings.Join(smc.commands, "|"))

	return help.Result{Text: text}
}

func (smc *SetModelHandler) isModelSupported(ctx context.Context, modelName string) (bool, error) {
	supportedModelIDs, err := smc.getSupportedModelIDs(ctx)
	if err != nil {
		return false, err
	}

	for _, supportedModelID := range supportedModelIDs {
		if modelName == supportedModelID {
			return true, nil
		}
	}

	return false, nil
}

type GetModelsCommand struct {
	ModelCommand
	command       string
	loader        *Loader
	modeDetector  func() bool
	adminDetector func(req *msg.Request) bool
}

func NewGetModelsCommand(
	cfg *Config,
	db storage.Client,
	loader *Loader,
	modeDetector func() bool,
	adminDetector func(req *msg.Request) bool,
) *GetModelsCommand {
	return &GetModelsCommand{
		ModelCommand: ModelCommand{
			cfg: cfg,
			db:  db,
		},
		command:       "/models",
		loader:        loader,
		modeDetector:  modeDetector,
		adminDetector: adminDetector,
	}
}

func (gmc *GetModelsCommand) CanHandle(_ context.Context, req *msg.Request) (bool, error) {
	if !utils.MatchesCommand(req.Message, gmc.command) {
		return false, nil
	}

	if gmc.modeDetector() && !gmc.adminDetector(req) {
		return false, nil
	}

	return true, nil
}

func (gmc *GetModelsCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will get list of supported ChatGPT models")

	modelIDs, err := gmc.getSupportedModelIDs(ctx)
	if err != nil {
		return nil, err
	}

	currentModel := gmc.loader.LoadModel(ctx, req)

	opts := &msg.Options{}
	opts.WithIsTempPredefinedResponse()
	opts.WithFormat(msg.OutputFormatHTML)

	sort.Strings(modelIDs)
	for i, modelID := range modelIDs {
		if strings.HasPrefix(modelID, "gpt-") {
			opts.WithPredefinedResponse(msg.PredefinedResponse{
				Text: fmt.Sprintf("/model %s", modelID),
				Type: msg.PredefinedResponseOutline,
			})
		}

		if modelID == currentModel.GetName() {
			modelIDs[i] = "<b>" + modelID + "</b> [current]"
		}
	}

	return &msg.Response{
		Message: fmt.Sprintf(`<b>Supported ChatGPT models:</b>
%s
`, strings.Join(modelIDs, "\n")),
		Type:    msg.Success,
		Options: opts,
	}, nil
}

func (gmc *GetModelsCommand) GetHelp(context.Context, *msg.Request) help.Result {
	text := fmt.Sprintf("%s: to get the list of supported ChatGPT models", gmc.command)

	return help.Result{Text: text, PredefinedOption: gmc.command}
}
