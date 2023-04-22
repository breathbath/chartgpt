package chartgpt

import (
	"breathbathChartGPT/pkg/msg"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"

	logging "github.com/sirupsen/logrus"
)

const URL = "https://api.openai.com"
const CompletionsURL = URL + "/v1/chat/completions"

type Handler struct {
	cfg *Config
}

func NewHandler(cfg *Config) (*Handler, error) {
	err := cfg.Validate()
	if err.HasErrors() {
		return nil, err
	}

	return &Handler{
		cfg: cfg,
	}, nil
}

func (h *Handler) Handle(ctx context.Context, req *msg.Request) (*msg.Response, error) {
	log := logging.WithContext(ctx)

	requestData := map[string]interface{}{
		"model": h.cfg.Model,
		"messages": []map[string]interface{}{
			{
				"role":    h.cfg.Role,
				"content": req.Message,
			},
		},
	}

	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}

	method := "POST"

	httpReq, err := http.NewRequestWithContext(ctx, method, CompletionsURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}

	log.Infof("will do chartgpt request, url: %q, method: %s, body: %q", CompletionsURL, method, requestBody)

	httpReq.Header.Set("Authorization", "Bearer "+h.cfg.ApiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Warnf("failed to dump response: %v", err)
	} else {
		log.Infof("response: %q", string(dump))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, errors.New("bad response code from ChartGPT")
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read response body: %v", err)
		return nil, errors.New("failed to read ChartGPT response")
	}

	chartResp := new(ChatCompletionResponse)
	err = json.Unmarshal(responseBody, chartResp)
	if err != nil {
		log.Errorf("failed to pack response data into ChatCompletionResponse model: %v", err)
		return nil, errors.New("failed to interpret ChartGPT response")
	}

	respToGive := &msg.Response{
		Messages: []msg.ResponseMessage{},
	}
	for _, choice := range chartResp.Choices {
		if choice.Message.Content == "" {
			continue
		}

		respToGive.Messages = append(respToGive.Messages, msg.ResponseMessage{
			Message: choice.Message.Content,
			Type:    msg.Success,
			Meta: map[string]interface{}{
				"created": chartResp.CreatedAt,
			},
		})
	}

	if len(respToGive.Messages) == 0 {
		return &msg.Response{
			Messages: []msg.ResponseMessage{
				{
					Message: "failed to interpret ChartGPT response",
					Type:    msg.Error,
				},
			},
		}, nil
	}

	return respToGive, nil
}
