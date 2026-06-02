// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func deepseekMeta() PluginMetadata {
	return PluginMetadata{
		ID:            "com.morcherlf.deepseek",
		Name:          I18n("DeepSeek AI"),
		Version:       "0.1.0",
		Description:   I18n("Provides AI chat model capability via the DeepSeek official API."),
		MainServiceID: "deepseek-service",
		Author:        "MorCherlf",
		Site:          "https://github.com/MorCherlf/Griffino-DeepSeek-Plugin",
		Tutorial:      "https://platform.deepseek.com/api-docs",
		Internationalization: &PluginInternationalization{
			DefaultLanguage:    "zh-CN",
			SupportedLanguages: []string{"zh-CN", "en-US"},
			LocalizationFiles: map[string]string{
				"zh-CN": "i18n/zh_CN.json",
				"en-US": "i18n/en_US.json",
			},
		},
		PermissionsRequested: []PluginPermission{
			{
				Name:        "network.outbound.internet",
				Description: I18n("Requires internet access to call the DeepSeek API."),
			},
		},
	}
}

func deepseekBootDef(t *testing.T) ConfigModelDefinition {
	t.Helper()
	type bootCfg struct {
		APIKey  string `griffino:"key=DEEPSEEK_API_KEY,type=password,name=DeepSeek API Key,description=DeepSeek platform API key.,optional=false,group=API Configuration"`
		Model   string `griffino:"key=DEEPSEEK_MODEL,type=options,name=Default Model,description=DeepSeek model used for chat requests.,optional=false,default=deepseek-chat,group=API Configuration,values=deepseek-chat:DeepSeek-V3 (Recommended)|deepseek-reasoner:DeepSeek-R1 (Reasoning)"`
		BaseURL string `griffino:"key=DEEPSEEK_BASE_URL,type=string,name=API Base URL,description=Base URL for DeepSeek compatible APIs.,optional=true,default=https://api.deepseek.com,group=API Configuration"`
		Tokens  int    `griffino:"key=MAX_TOKENS,type=int,name=Max Tokens,description=Maximum completion tokens per request.,optional=true,default=4096,validation=min:256;max:8192,group=Model Parameters"`
		Prompt  string `griffino:"key=SYSTEM_PROMPT,type=string,name=System Prompt,description=Global system prompt applied to all users.,optional=true,default=You are a helpful assistant.,group=Model Parameters"`
	}
	def, err := ParseBootConfig(bootCfg{})
	if err != nil {
		t.Fatalf("ParseBootConfig: %v", err)
	}
	return def
}

func TestGenerateManifest(t *testing.T) {
	meta := deepseekMeta()
	timeout := 60000
	providers := []ProviderRegistration{
		{
			CapabilityID:         "ai_chat_model",
			CapabilityType:       "ai.chat.model",
			Name:                 I18n("AI Chat Model"),
			Description:          I18n("Provides AI chat completion via DeepSeek API."),
			StandardInterfaceRef: "griffino.interfaces.ai.chat@1.0.0",
			TimeoutMS:            &timeout,
		},
	}

	manifest := GenerateManifest(meta, providers, nil, nil, true, true)

	if manifest["griffinoPluginManifestVersion"] != "1.0" {
		t.Errorf("manifest version = %v", manifest["griffinoPluginManifestVersion"])
	}
	if manifest["id"] != "com.morcherlf.deepseek" {
		t.Errorf("id = %v", manifest["id"])
	}

	caps := manifest["capabilities"].([]map[string]any)
	if len(caps) != 1 {
		t.Fatalf("got %d capabilities, want 1", len(caps))
	}
	cap0 := caps[0]
	if cap0["role"] != "provider" || cap0["type"] != "ai.chat.model" {
		t.Errorf("capability shape = %+v", cap0)
	}
	entry := cap0["entryPoint"].(map[string]any)
	details := entry["details"].(map[string]any)
	wantTopic := "invoke.com.morcherlf.deepseek.ai_chat_model.v1"
	if details["requestTopicPattern"] != wantTopic {
		t.Errorf("requestTopicPattern = %v, want %v", details["requestTopicPattern"], wantTopic)
	}
	if cap0["standardInterfaceRef"] != "griffino.interfaces.ai.chat@1.0.0" {
		t.Errorf("standardInterfaceRef = %v", cap0["standardInterfaceRef"])
	}
	if cap0["defaultTimeoutMilliseconds"] != 60000 {
		t.Errorf("defaultTimeoutMilliseconds = %v", cap0["defaultTimeoutMilliseconds"])
	}

	wantConfigFiles := map[string]string{
		"bootConfig":  "config.boot.json",
		"runtimeBoot": "plugin.boot.yml",
		"userConfig":  "config.user.json",
	}
	if !reflect.DeepEqual(manifest["configurationFiles"], wantConfigFiles) {
		t.Errorf("configurationFiles = %+v, want %+v", manifest["configurationFiles"], wantConfigFiles)
	}

	// site always present.
	if manifest["site"] != meta.Site {
		t.Errorf("site = %v", manifest["site"])
	}
	// generator does NOT emit a dependencies block.
	if _, ok := manifest["dependencies"]; ok {
		t.Errorf("manifest should not contain dependencies")
	}
}

func TestGenerateManifestSiteDefaultsEmpty(t *testing.T) {
	meta := PluginMetadata{ID: "x", Name: I18n("X"), Version: "1.0"}
	manifest := GenerateManifest(meta, nil, nil, nil, false, false)
	if manifest["site"] != "" {
		t.Errorf("site = %v, want empty", manifest["site"])
	}
	if _, ok := manifest["configurationFiles"]; ok {
		t.Errorf("no config files expected")
	}
}

func TestGenerateBootConfig(t *testing.T) {
	meta := deepseekMeta()
	def := deepseekBootDef(t)

	cfg := GenerateBootConfig(meta, def)

	if cfg["GriffinoPluginConfigVersion"] != "1.0" {
		t.Errorf("version = %v", cfg["GriffinoPluginConfigVersion"])
	}
	if cfg["name"] != "DeepSeek AI" {
		t.Errorf("name = %v, want plain string", cfg["name"])
	}

	services := cfg["services"].([]map[string]any)
	if len(services) != 1 {
		t.Fatalf("got %d services, want 1", len(services))
	}
	if services[0]["id"] != "deepseek-service" {
		t.Errorf("service id = %v", services[0]["id"])
	}
	configs := services[0]["configs"].([]map[string]any)
	if len(configs) != 5 {
		t.Fatalf("got %d configs, want 5", len(configs))
	}

	apiKey := configs[0]
	wantAPIKey := map[string]any{
		"key":         "DEEPSEEK_API_KEY",
		"type":        "password",
		"name":        "DeepSeek API Key",
		"description": "DeepSeek platform API key.",
		"optional":    false,
		"group":       "API Configuration",
	}
	if !reflect.DeepEqual(apiKey, wantAPIKey) {
		t.Errorf("apiKey entry = %+v, want %+v", apiKey, wantAPIKey)
	}

	model := configs[1]
	if model["default"] != "deepseek-chat" {
		t.Errorf("model default = %v", model["default"])
	}
	values := model["values"].([]map[string]any)
	if len(values) != 2 || values[0]["value"] != "deepseek-chat" || values[0]["display"] != "DeepSeek-V3 (Recommended)" {
		t.Errorf("model values = %+v", values)
	}

	tokens := configs[3]
	if tokens["type"] != "int" || tokens["default"] != 4096 {
		t.Errorf("tokens entry = %+v", tokens)
	}
	wantValidation := map[string]any{"minimum": 256, "maximum": 8192}
	if !reflect.DeepEqual(tokens["validation"], wantValidation) {
		t.Errorf("tokens validation = %+v", tokens["validation"])
	}
}

func TestGenerateUserConfig(t *testing.T) {
	type userCfg struct {
		Prompt string  `griffino:"key=USER_SYSTEM_PROMPT,type=string,name=Personal System Prompt,description=Prompt appended after the global system prompt.,optional=true,default=,group=Personalization"`
		Temp   float64 `griffino:"key=TEMPERATURE,type=float,name=Temperature,description=Controls response randomness.,optional=true,default=1.0,group=Personalization"`
	}
	def, err := ParseUserConfig(userCfg{})
	if err != nil {
		t.Fatalf("ParseUserConfig: %v", err)
	}
	cfg := GenerateUserConfig(deepseekMeta(), def)

	if _, ok := cfg["services"]; ok {
		t.Errorf("user config must not wrap fields in services")
	}
	configs := cfg["configs"].([]map[string]any)
	if len(configs) != 2 {
		t.Fatalf("got %d configs, want 2", len(configs))
	}
	if configs[0]["default"] != "" {
		t.Errorf("empty-string default should be present and empty, got %v", configs[0]["default"])
	}
	if configs[1]["default"] != 1.0 {
		t.Errorf("temp default = %v", configs[1]["default"])
	}
}

func TestGenerateBootYAMLData(t *testing.T) {
	def := deepseekBootDef(t)
	data, err := GenerateBootYAMLData(deepseekMeta(), def)
	if err != nil {
		t.Fatalf("GenerateBootYAMLData: %v", err)
	}
	services := data["services"].(map[string]any)
	svc := services["deepseek-service"].(map[string]any)
	env := svc["environment"].([]string)
	if len(env) != len(SYSTEM_ENVIRONMENT_VARIABLES)+5 {
		t.Fatalf("env len = %d", len(env))
	}
	if env[0] != "RABBITMQ_HOST={{system.rabbitmq.host}}" {
		t.Errorf("first env = %q", env[0])
	}
	if env[len(env)-1] != "SYSTEM_PROMPT" {
		t.Errorf("last env = %q", env[len(env)-1])
	}

	// requires MainServiceID
	noSvc := deepseekMeta()
	noSvc.MainServiceID = ""
	if _, err := GenerateBootYAMLData(noSvc, def); err == nil {
		t.Error("expected error when MainServiceID is empty")
	}
}

func TestWriteJSONFileWithBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.boot.json")

	if err := WriteJSONFile(path, map[string]any{"a": 1}); err != nil {
		t.Fatalf("write: %v", err)
	}
	first, _ := os.ReadFile(path)
	if want := "{\n  \"a\": 1\n}\n"; string(first) != want {
		t.Errorf("content = %q, want %q", string(first), want)
	}

	// Second write backs up the original to .1
	if err := WriteJSONFile(path, map[string]any{"b": 2}); err != nil {
		t.Fatalf("write 2: %v", err)
	}
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("expected backup at %s.1: %v", path, err)
	}
	backup, _ := os.ReadFile(path + ".1")
	if string(backup) != string(first) {
		t.Errorf("backup content mismatch")
	}
}

func TestWriteBootYAMLFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plugin.boot.yml")
	def := deepseekBootDef(t)
	if err := WriteBootYAMLFile(path, deepseekMeta(), def); err != nil {
		t.Fatalf("WriteBootYAMLFile: %v", err)
	}
	content, _ := os.ReadFile(path)
	s := string(content)
	for _, want := range []string{
		`pluginBootSpecVersion: "1.0"`,
		`pluginId: "com.morcherlf.deepseek"`,
		`mainServiceId: "deepseek-service"`,
		"  deepseek-service:",
		"    build_instructions:",
		`      - "RABBITMQ_HOST={{system.rabbitmq.host}}"`,
		`      - "DEEPSEEK_API_KEY"`,
		`      - "SYSTEM_PROMPT"`,
		"    ports: []",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("boot yaml missing %q\n---\n%s", want, s)
		}
	}
}
