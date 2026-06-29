package llm

// ── 请求 wire 结构 ──

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []wireMessage `json:"messages"`
}

type wireMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ── 响应 wire 结构 ──

type chatResponse struct {
	Choices []choice `json:"choices"`
}

type choice struct {
	Message respMessage `json:"message"`
}

type respMessage struct {
	Content string `json:"content"`
}
