package chartgpt

type ChatCompletionResponse struct {
	Id         string                   `json:"id"`
	Object     string                   `json:"object"`
	CreatedAt  int64                    `json:"created"`
	Model      string                   `json:"model"`
	Choices    []ChatCompletionChoice   `json:"choices"`
	Prompt     string                   `json:"prompt"`
	Completion ChatCompletionCompletion `json:"completion"`
	Usage      ChatCompletionUsage      `json:"usage"`
}

type ChatCompletionChoice struct {
	Text         string                 `json:"text"`
	Index        int                    `json:"index"`
	Logprobs     ChatCompletionLogprobs `json:"logprobs"`
	FinishReason string                 `json:"finish_reason"`
	Message      ChatCompletionMessage  `json:"message"`
}

type ChatCompletionLogprobs struct {
	Tokens        []string      `json:"tokens"`
	TokenLogprobs []float64     `json:"token_logprobs"`
	TopLogprobs   []interface{} `json:"top_logprobs"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionCompletion struct {
	Id             string                 `json:"id"`
	CreatedAt      int64                  `json:"created"`
	Model          string                 `json:"model"`
	Choices        []ChatCompletionChoice `json:"choices"`
	Prompt         string                 `json:"prompt"`
	Text           string                 `json:"text"`
	FinishReason   string                 `json:"finish_reason"`
	SelectedAnswer int                    `json:"selected_answer"`
}

type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
