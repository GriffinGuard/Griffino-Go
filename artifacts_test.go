// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteArtifacts(t *testing.T) {
	type bootCfg struct {
		APIKey string `griffino:"key=DEEPSEEK_API_KEY,type=password,name=DeepSeek API Key,description=DeepSeek platform API key.,optional=false,group=API Configuration"`
		Tokens int    `griffino:"key=MAX_TOKENS,type=int,name=Max Tokens,description=Maximum completion tokens per request.,optional=true,default=4096,validation=min:256;max:8192,group=Model Parameters"`
	}
	type userCfg struct {
		Temperature float64 `griffino:"key=TEMPERATURE,type=float,name=Temperature,description=Controls response randomness.,optional=true,default=1.0,group=Personalization"`
	}

	client, err := New(WithMetadata(PluginMetadata{
		ID:            "com.morcherlf.deepseek",
		Name:          I18n("DeepSeek AI"),
		Version:       "0.1.0",
		Description:   I18n("Provides AI chat model capability via the DeepSeek official API."),
		MainServiceID: "deepseek-service",
		Author:        "MorCherlf",
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	timeout := 60000
	client.Provider(ProviderRegistration{
		CapabilityID:   "ai_chat_model",
		CapabilityType: "ai.chat.model",
		Name:           I18n("AI Chat Model"),
		Description:    I18n("Provides AI chat completion via DeepSeek API."),
		TimeoutMS:      &timeout,
		Handler: func(ctx *HandlerContext) (map[string]any, error) {
			return nil, nil
		},
	})

	dir := t.TempDir()
	if err := WriteArtifacts(client, dir, &bootCfg{}, &userCfg{}); err != nil {
		t.Fatalf("WriteArtifacts: %v", err)
	}

	for _, name := range []string{
		manifestFileName, bootConfigFileName, userConfigFileName, bootYAMLFileName,
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected %s to exist: %v", name, err)
		}
	}

	manifest := decodeJSON(t, filepath.Join(dir, manifestFileName))
	if manifest["id"] != "com.morcherlf.deepseek" {
		t.Errorf("manifest id = %v, want com.morcherlf.deepseek", manifest["id"])
	}
	caps, ok := manifest["capabilities"].([]any)
	if !ok || len(caps) != 1 {
		t.Fatalf("manifest capabilities = %v, want 1 entry", manifest["capabilities"])
	}
	cap0 := caps[0].(map[string]any)
	if cap0["id"] != "ai_chat_model" || cap0["type"] != "ai.chat.model" {
		t.Errorf("capability = %v, want ai_chat_model / ai.chat.model", cap0)
	}
	cfgFiles := manifest["configurationFiles"].(map[string]any)
	if cfgFiles["bootConfig"] != bootConfigFileName || cfgFiles["userConfig"] != userConfigFileName {
		t.Errorf("configurationFiles = %v", cfgFiles)
	}

	boot := decodeJSON(t, filepath.Join(dir, bootConfigFileName))
	if boot["pluginId"] != "com.morcherlf.deepseek" {
		t.Errorf("boot pluginId = %v", boot["pluginId"])
	}
	services, ok := boot["services"].([]any)
	if !ok || len(services) != 1 {
		t.Fatalf("boot services = %v, want 1 entry", boot["services"])
	}
	svc := services[0].(map[string]any)
	if svc["id"] != "deepseek-service" {
		t.Errorf("boot service id = %v, want deepseek-service", svc["id"])
	}
	if configs, ok := svc["configs"].([]any); !ok || len(configs) != 2 {
		t.Errorf("boot configs = %v, want 2 entries", svc["configs"])
	}

	user := decodeJSON(t, filepath.Join(dir, userConfigFileName))
	if configs, ok := user["configs"].([]any); !ok || len(configs) != 1 {
		t.Errorf("user configs = %v, want 1 entry", user["configs"])
	}
}

func decodeJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return out
}
