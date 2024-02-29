package msg

import (
	"encoding/json"
	"time"
)

type OutputFormat uint

const (
	OutputFormatUndefined OutputFormat = iota
	OutputFormatMarkdown1
	OutputFormatMarkdown2
	OutputFormatHTML
)

type DelayType uint

const (
	DelayTypeMessage DelayType = iota + 1
	DelayTypeCallback
)

const (
	PredefinedResponseOutline = "outline"
	PredefinedResponseInline  = "inline"
)

type PredefinedResponse struct {
	Text string
	Data string
	Type string
	Link string
}

type PredefinedResponseOptions struct {
	Responses []PredefinedResponse
	IsTemp    bool
}

type DelayedOptions struct {
	Timeout         time.Duration
	Key             string
	CallbackPayload json.RawMessage
	DelayType       DelayType
}

type Options struct {
	OutputFormat              OutputFormat
	IsRespToHiddenMessage     bool
	PredefinedResponseOptions *PredefinedResponseOptions
}

func (o *Options) WithFormat(f OutputFormat) {
	o.OutputFormat = f
}

func (o *Options) WithIsResponseToHiddenMessage() {
	o.IsRespToHiddenMessage = true
}

func (o *Options) WithPredefinedResponse(resp PredefinedResponse) {
	if o.PredefinedResponseOptions == nil {
		o.PredefinedResponseOptions = &PredefinedResponseOptions{}
	}

	if o.PredefinedResponseOptions.Responses == nil {
		o.PredefinedResponseOptions.Responses = []PredefinedResponse{}
	}

	o.PredefinedResponseOptions.Responses = append(o.PredefinedResponseOptions.Responses, resp)
}

func (o *Options) WithIsTempPredefinedResponse() {
	if o.PredefinedResponseOptions == nil {
		o.PredefinedResponseOptions = &PredefinedResponseOptions{}
	}
	o.PredefinedResponseOptions.IsTemp = true
}

func (o *Options) GetFormat() OutputFormat {
	if o == nil {
		return OutputFormatUndefined
	}

	return o.OutputFormat
}

func (o *Options) IsResponseToHiddenMessage() bool {
	if o == nil {
		return false
	}

	return o.IsRespToHiddenMessage
}

func (o *Options) GetPredefinedResponses() []PredefinedResponse {
	if o == nil || o.PredefinedResponseOptions == nil {
		return nil
	}

	return o.PredefinedResponseOptions.Responses
}

func (o *Options) IsTempPredefinedResponse() bool {
	if o == nil || o.PredefinedResponseOptions == nil {
		return false
	}

	return o.PredefinedResponseOptions.IsTemp
}
