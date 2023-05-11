package chatgpt

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/rest"
	"breathbathChatGPT/pkg/storage"
	"breathbathChatGPT/pkg/utils"
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"sort"
	"strings"
	"time"
)

type SetModelCommand struct {
	ModelCommand
	operators []string
	loader    *Loader
}

func NewSetModelCommand(cfg *Config, db storage.Client, loader *Loader) *SetModelCommand {
	return &SetModelCommand{
		ModelCommand: ModelCommand{
			cfg: cfg,
			db:  db,
		},
		operators: []string{"setm", "setmodel"},
		loader:    loader,
	}
}

type ModelCommand struct {
	cfg *Config
	db  storage.Client
}

func (mc *ModelCommand) getSupportedModelIDs(ctx context.Context) ([]string, error) {
	modelsResp := new(ModelsResponse)
	reqsr := rest.NewRequester(ModelsURL, modelsResp)
	reqsr.WithBearer(mc.cfg.ApiKey)
	reqsr.WithCache("chatgpt/models", mc.db, time.Hour*24)

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

func (smc *SetModelCommand) CanHandle(ctx context.Context, req *msg.Request) bool {
	return utils.MatchesAny(req.Message, CommandPrefix, smc.operators)
}

func (smc *SetModelCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	modelName := extractCommandValue(req.Message, smc.operators)
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

func (smc *SetModelCommand) GetHelp() string {
	operatorHelp := buildHelpFromOperators(smc.operators)

	return fmt.Sprintf("%s #modelName#: to change the active ChatGPT model", operatorHelp)
}

func (smc *SetModelCommand) isModelSupported(ctx context.Context, modelName string) (bool, error) {
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
	operators []string
	loader    *Loader
}

func NewGetModelsCommand(cfg *Config, db storage.Client, loader *Loader) *GetModelsCommand {
	return &GetModelsCommand{
		ModelCommand: ModelCommand{
			cfg: cfg,
			db:  db,
		},
		operators: []string{"getm", "models"},
		loader:    loader,
	}
}

func (gmc *GetModelsCommand) CanHandle(ctx context.Context, req *msg.Request) bool {
	return utils.MatchesAny(req.Message, CommandPrefix, gmc.operators)
}

func (gmc *GetModelsCommand) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logrus.WithContext(ctx)

	log.Debug("will get list of supported ChatGPT models")

	modelIDs, err := gmc.getSupportedModelIDs(ctx)
	if err != nil {
		return nil, err
	}

	currentModel := gmc.loader.LoadModel(ctx, req)

	sort.Strings(modelIDs)
	for i, modelId := range modelIDs {
		if modelId == currentModel.GetName() {
			modelIDs[i] = "<b>" + modelId + "</b> [current]"
		}
	}

	return &msg.Response{
		Message: fmt.Sprintf(`<b>Supported ChatGPT models:</b>
%s
`, strings.Join(modelIDs, "\n")),
		Type: msg.Success,
		Meta: map[string]interface{}{"format": "html"},
	}, nil
}

func (gmc *GetModelsCommand) GetHelp() string {
	operatorHelp := buildHelpFromOperators(gmc.operators)

	return fmt.Sprintf("%s: to get the list of supported ChatGPT models", operatorHelp)
}
