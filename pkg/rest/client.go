package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"time"

	"breathbathChatGPT/pkg/storage"

	"github.com/pkg/errors"
	logging "github.com/sirupsen/logrus"
)

const defaultRequestCacheValidity = time.Hour

type Requester struct {
	method        string
	url           string
	apiKey        string
	input         interface{}
	output        interface{}
	db            storage.Client
	cacheValidity time.Duration
	cacheKey      string
}

func NewRequester(url string, target interface{}) *Requester {
	return &Requester{
		method:        http.MethodGet,
		url:           url,
		output:        target,
		cacheValidity: defaultRequestCacheValidity,
	}
}

func (r *Requester) WithMethod(m string) {
	r.method = m
}

func (r *Requester) WithPOST() {
	r.method = http.MethodPost
}

func (r *Requester) WithCache(key string, c storage.Client, validity time.Duration) {
	r.db = c
	if validity > 0 {
		r.cacheValidity = validity
	}
	r.cacheKey = key
}

func (r *Requester) WithInput(i interface{}) {
	r.input = i
}

func (r *Requester) WithBearer(key string) {
	r.apiKey = key
}

func (r *Requester) addHeaders(httpReq *http.Request) {
	if r.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+r.apiKey)
	}

	httpReq.Header.Set("Content-Type", "application/json")
}

func (r *Requester) getCacheKey() string {
	if r.cacheKey == "" {
		return r.url
	}

	return r.cacheKey
}

func (r *Requester) Request(ctx context.Context) error {
	log := logging.WithContext(ctx)

	if r.db != nil {
		found, err := r.db.Load(ctx, r.getCacheKey(), r.output)
		if err != nil {
			return err
		}

		if found {
			return nil
		}
	}

	var bodyReader io.Reader
	var requestBody []byte
	var err error
	if r.input != nil {
		requestBody, err = json.Marshal(r.input)
		if err != nil {
			return errors.Wrap(err, "failed to create an http request body")
		}

		bodyReader = bytes.NewBuffer(requestBody)
	}

	httpReq, err := http.NewRequestWithContext(ctx, r.method, r.url, bodyReader)
	if err != nil {
		return errors.Wrap(err, "failed to create an http request")
	}

	if len(requestBody) > 0 {
		log.Debugf("http request, url: %q, method: %s, body: %q", r.url, r.method, string(requestBody))
	} else {
		log.Debugf("will do chatgpt request, url: %q, method: %s", r.url, r.method)
	}

	r.addHeaders(httpReq)

	err = r.request(ctx, httpReq, r.output)
	if err != nil {
		return err
	}

	if r.db != nil {
		err := r.db.Save(ctx, r.getCacheKey(), r.output, r.cacheValidity)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Requester) request(ctx context.Context, httpReq *http.Request, target interface{}) (err error) {
	log := logging.WithContext(ctx)

	client := &http.Client{}

	resp, err := client.Do(httpReq)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	dump, err := httputil.DumpResponse(resp, true)
	if err != nil {
		log.Warnf("failed to dump response: %v", err)
	} else {
		log.Infof("response: %q", string(dump))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("bad response code from ChatGPT")
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read response body: %v", err)
		return errors.New("failed to read ChatGPT response")
	}

	err = json.Unmarshal(responseBody, target)
	if err != nil {
		log.Errorf("failed to pack response data into ChatCompletionResponse model: %v", err)
		return errors.New("failed to interpret ChatGPT response")
	}

	return nil
}
