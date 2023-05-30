package chatgpt

import "encoding/json"

type ChatCompletionResponse struct {
	ID         string                   `json:"id"`
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

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionCompletion struct {
	ID             string                 `json:"id"`
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

type ConfiguredModel struct {
	Model string `json:"model"`
}

func (m *ConfiguredModel) GetName() string {
	if m == nil {
		return ""
	}

	return m.Model
}

type ModelsResponse struct {
	Models []Model `json:"data"`
	Object string  `json:"object"`
}

type Model struct {
	ID         string          `json:"id"`
	Object     string          `json:"object"`
	Owner      string          `json:"owned_by"`
	Permission json.RawMessage `json:"permission"`
}

type ConversationMessage struct {
	Role      Role
	Text      string
	CreatedAt int64
}

type Context struct {
	Message            string
	CreatedAtTimestamp int64
}

func (c *Context) GetMessage() string {
	if c == nil {
		return ""
	}

	return c.Message
}

func (c *Context) GetCreatedAt() int64 {
	if c == nil {
		return 0
	}

	return c.CreatedAtTimestamp
}

type Conversation struct {
	ID       string
	Context  *Context
	Messages []ConversationMessage
}

func (c Conversation) ToRaw() []map[string]interface{} {
	convResp := make([]map[string]interface{}, 0, len(c.Messages)+1)
	if c.Context.GetMessage() != "" {
		convResp = append(convResp, map[string]interface{}{
			"role":    RoleSystem,
			"content": c.Context.GetMessage(),
		})
	}

	for _, convMsg := range c.Messages {
		convResp = append(convResp, map[string]interface{}{
			"role":    convMsg.Role,
			"content": convMsg.Text,
		})
	}

	return convResp
}
