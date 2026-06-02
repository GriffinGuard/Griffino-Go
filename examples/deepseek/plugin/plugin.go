// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

// Package plugin holds the shared definitions for the DeepSeek example: the
// plugin metadata, the boot/user config structs, and a Client builder. Both the
// runtime entry point (../main.go) and the artifact generator (../gen/main.go)
// import this package so they describe exactly the same plugin.
package plugin

import (
	"fmt"

	"github.com/GriffinGuard/Griffino-Go"
)

// BootConfig is the admin-set boot configuration for the plugin. Each field's
// `griffino:"..."` tag describes how it is rendered into config.boot.json.
// These mirror the fields of the reference Python example.
type BootConfig struct {
	APIKey  string `griffino:"key=DEEPSEEK_API_KEY,type=password,name=DeepSeek API Key,description=DeepSeek platform API key.,optional=false,group=API Configuration"`
	Model   string `griffino:"key=DEEPSEEK_MODEL,type=options,name=Default Model,description=DeepSeek model used for chat requests.,optional=false,default=deepseek-chat,group=API Configuration,values=deepseek-chat:DeepSeek-V3 (Recommended)|deepseek-reasoner:DeepSeek-R1 (Reasoning)"`
	BaseURL string `griffino:"key=DEEPSEEK_BASE_URL,type=string,name=API Base URL,description=Base URL for DeepSeek compatible APIs.,optional=true,default=https://api.deepseek.com,group=API Configuration"`
	Tokens  int    `griffino:"key=MAX_TOKENS,type=int,name=Max Tokens,description=Maximum completion tokens per request.,optional=true,default=4096,validation=min:256;max:8192,group=Model Parameters"`
	Prompt  string `griffino:"key=SYSTEM_PROMPT,type=string,name=System Prompt,description=Global system prompt applied to all users.,optional=true,default=You are a helpful assistant.,group=Model Parameters"`
}

// UserConfig is the per-user configuration, rendered into config.user.json.
type UserConfig struct {
	UserSystemPrompt string  `griffino:"key=USER_SYSTEM_PROMPT,type=string,name=Personal System Prompt,description=Prompt appended after the global system prompt.,optional=true,default=,group=Personalization"`
	Temperature      float64 `griffino:"key=TEMPERATURE,type=float,name=Temperature,description=Controls response randomness.,optional=true,default=1.0,group=Personalization"`
	ReplyLanguage    string  `griffino:"key=REPLY_LANGUAGE,type=options,name=Reply Language,description=Preferred reply language.,optional=true,default=auto,group=Personalization,values=auto:Automatic|zh-CN:Simplified Chinese|en-US:English"`
}

// Metadata returns the plugin's static identity and presentation data.
func Metadata() griffino.PluginMetadata {
	return griffino.PluginMetadata{
		ID:                "com.morcherlf.deepseek",
		Name:              griffino.I18n("DeepSeek AI"),
		Version:           "0.1.0",
		Description:       griffino.I18n("Provides AI chat model capability via the DeepSeek official API."),
		MainServiceID:     "deepseek-service",
		Author:            "MorCherlf",
		License:           "MIT",
		Site:              "https://github.com/MorCherlf/Griffino-DeepSeek-Plugin",
		Tutorial:          "https://platform.deepseek.com/api-docs",
		MainSystemVersion: ">=0.1.0",
		Internationalization: &griffino.PluginInternationalization{
			DefaultLanguage:    "zh-CN",
			SupportedLanguages: []string{"zh-CN", "en-US"},
			LocalizationFiles: map[string]string{
				"zh-CN": "i18n/zh_CN.json",
				"en-US": "i18n/en_US.json",
			},
		},
		PermissionsRequested: []griffino.PluginPermission{
			{
				Name:        "network.outbound.internet",
				Description: griffino.I18n("Requires internet access to call the DeepSeek API."),
			},
		},
	}
}

// NewClient constructs the example Client and registers its single capability
// provider. The provided options are appended after WithMetadata, so callers
// (such as the generator) can inject a transport or explicit config.
func NewClient(opts ...griffino.Option) (*griffino.Client, error) {
	opts = append([]griffino.Option{griffino.WithMetadata(Metadata())}, opts...)
	client, err := griffino.New(opts...)
	if err != nil {
		return nil, err
	}

	timeout := 60000
	client.Provider(griffino.ProviderRegistration{
		CapabilityID:         "ai_chat_model",
		CapabilityType:       "ai.chat.model",
		Name:                 griffino.I18n("AI Chat Model"),
		Description:          griffino.I18n("Provides AI chat completion via DeepSeek API."),
		StandardInterfaceRef: "griffino.interfaces.ai.chat@1.0.0",
		TimeoutMS:            &timeout,
		Handler:              handleChat,
	})

	return client, nil
}

// handleChat handles an ai.chat.model invocation.
//
// This is an illustrative stub: it validates the incoming payload and returns a
// canned completion. A real plugin would read its boot/user configuration, call
// the DeepSeek API (an OpenAI-compatible chat completions endpoint), and return
// the model's response here.
func handleChat(ctx *griffino.HandlerContext) (map[string]any, error) {
	messages, ok := ctx.Payload["messages"].([]any)
	if !ok {
		return nil, fmt.Errorf("payload.messages must be a list")
	}

	// A real implementation would build the final message list from the boot
	// and user system prompts, then forward it to the DeepSeek API. Here we
	// echo back a stub completion so the example is runnable without a key.
	_ = messages

	return map[string]any{
		"content": "This is a stub completion from the Griffino DeepSeek example.",
		"model":   "deepseek-chat",
		"usage": map[string]any{
			"promptTokens":     0,
			"completionTokens": 0,
			"totalTokens":      0,
		},
	}, nil
}
