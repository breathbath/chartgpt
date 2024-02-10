package msg

type OutputFormat uint

const (
	OutputFormatUndefined OutputFormat = iota
	OutputFormatMarkdown1
	OutputFormatMarkdown2
	OutputFormatHTML
)

const (
	PredefinedResponseOutline = "outline"
	PredefinedResponseInline  = "inline"
)

type PredefinedResponse struct {
	Text string
	Data string
	Type string
}

type PredefinedResponseOptions struct {
	Responses []PredefinedResponse
	IsTemp    bool
}

type Options struct {
	outputFormat              OutputFormat
	isResponseToHiddenMessage bool
	predefinedResponseOptions *PredefinedResponseOptions
}

func (o *Options) WithFormat(f OutputFormat) {
	o.outputFormat = f
}

func (o *Options) WithIsResponseToHiddenMessage() {
	o.isResponseToHiddenMessage = true
}

func (o *Options) WithPredefinedResponse(resp PredefinedResponse) {
	if o.predefinedResponseOptions == nil {
		o.predefinedResponseOptions = &PredefinedResponseOptions{}
	}

	if o.predefinedResponseOptions.Responses == nil {
		o.predefinedResponseOptions.Responses = []PredefinedResponse{}
	}

	o.predefinedResponseOptions.Responses = append(o.predefinedResponseOptions.Responses, resp)
}

func (o *Options) WithIsTempPredefinedResponse() {
	if o.predefinedResponseOptions == nil {
		o.predefinedResponseOptions = &PredefinedResponseOptions{}
	}
	o.predefinedResponseOptions.IsTemp = true
}

func (o *Options) GetFormat() OutputFormat {
	if o == nil {
		return OutputFormatUndefined
	}

	return o.outputFormat
}

func (o *Options) IsResponseToHiddenMessage() bool {
	if o == nil {
		return false
	}

	return o.isResponseToHiddenMessage
}

func (o *Options) GetPredefinedResponses() []PredefinedResponse {
	if o == nil || o.predefinedResponseOptions == nil {
		return nil
	}

	return o.predefinedResponseOptions.Responses
}

func (o *Options) IsTempPredefinedResponse() bool {
	if o == nil || o.predefinedResponseOptions == nil {
		return false
	}

	return o.predefinedResponseOptions.IsTemp
}
