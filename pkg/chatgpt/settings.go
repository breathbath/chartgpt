package chatgpt

import (
	"context"
	"time"

	"breathbathChatGPT/pkg/msg"
	"breathbathChatGPT/pkg/storage"

	"github.com/pkg/errors"
	logging "github.com/sirupsen/logrus"
)

const ModelChangeDuration = time.Minute * 10

type Loader struct {
	db  storage.Client
	cfg *Config
}

func NewSettingsLoader(db storage.Client, cfg *Config) *Loader {
	return &Loader{
		db:  db,
		cfg: cfg,
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
	return "chatgpt/model/" + req.GetConversationID()
}

func (l *Loader) SaveModel(ctx context.Context, m *ConfiguredModel, req *msg.Request) error {
	log := logging.WithContext(ctx)

	if m.Model == "" {
		return errors.New("empty model name for saving")
	}

	key := l.getModelKey(req)
	err := l.db.Save(ctx, key, m, ModelChangeDuration)
	if err != nil {
		return err
	}

	log.Debugf("saved current model setting, key: %q, valid till: %v, model: %q", key, ModelChangeDuration, m.Model)

	return nil
}

func (l *Loader) getDefaultModel() *ConfiguredModel {
	return &ConfiguredModel{
		Model: l.cfg.DefaultModel,
	}
}
