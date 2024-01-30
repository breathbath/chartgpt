package chatgpt

import (
	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"
	"context"

	"github.com/pkg/errors"
	logging "github.com/sirupsen/logrus"
)

const (
	modelVersion = "v1"
)

type Loader struct {
	db           storage.Client
	cfg          *Config
	isScopedMode func() bool
}

func NewSettingsLoader(db storage.Client, cfg *Config, isScopedMode func() bool) *Loader {
	return &Loader{
		db:           db,
		cfg:          cfg,
		isScopedMode: isScopedMode,
	}
}

func (l *Loader) LoadModel(ctx context.Context, req *msg.Request) *ConfiguredModel {
	log := logging.WithContext(ctx)

	m := new(ConfiguredModel)

	found, err := l.db.Load(ctx, l.getModelKey(req), m)
	if err != nil {
		log.Error(err)
		defaultModel := l.getDefaultModel()
		log.Debugf("falling back to default model name: %q", defaultModel.Model)
		return defaultModel
	}

	if !found {
		defaultModel := l.getDefaultModel()
		log.Debugf("falling back to default model name: %q", defaultModel.Model)
		return defaultModel
	}

	return m
}

func (l *Loader) getModelKey(req *msg.Request) string {
	if l.isScopedMode() {
		return storage.GenerateCacheKey(modelVersion, "chatgpt", "model_glob")
	}

	return storage.GenerateCacheKey(modelVersion, "chatgpt", "model", req.GetConversationID())
}

func (l *Loader) SaveModel(ctx context.Context, m *ConfiguredModel, req *msg.Request) error {
	log := logging.WithContext(ctx)

	if m.Model == "" {
		return errors.New("empty model name for saving")
	}

	key := l.getModelKey(req)
	err := l.db.Save(ctx, key, m, 0)
	if err != nil {
		return err
	}

	log.Debugf("saved current model setting, key: %q, model: %q", key, m.Model)

	return nil
}

func (l *Loader) getDefaultModel() *ConfiguredModel {
	return &ConfiguredModel{
		Model: l.cfg.DefaultModel,
	}
}
