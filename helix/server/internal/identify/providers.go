package identify

// ProviderPresets returns connection templates only. Model IDs are supplied
// by administrators after consulting the provider's current model listing.
func ProviderPresets() []ProviderConfig {
	return []ProviderConfig{
		{Name: "deepseek", Kind: "openai", BaseURL: "https://api.deepseek.com/v1"},
		{Name: "qwen", Kind: "openai", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1"},
		{Name: "claude", Kind: "anthropic", BaseURL: "https://api.anthropic.com"},
		{Name: "kimi", Kind: "openai", BaseURL: "https://api.moonshot.ai/v1"},
		{Name: "mimo", Kind: "openai", BaseURL: "https://api.xiaomimimo.com/v1", AuthHeader: "api-key"},
		{Name: "opencode-zen", Kind: "openai", BaseURL: "https://opencode.ai/zen/v1"},
		{Name: "openai-compatible", Kind: "openai"},
		{Name: "anthropic-compatible", Kind: "anthropic"},
	}
}
