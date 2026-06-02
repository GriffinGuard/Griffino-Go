// Copyright (c) 2024-2026 MorCherlf. Licensed under the MIT License.

package griffino

import (
	"reflect"
	"testing"
)

type sampleBootConfig struct {
	APIKey    string  `griffino:"key=DEEPSEEK_API_KEY,type=password,name=DeepSeek API Key,description=DeepSeek platform API key.,optional=false,group=API Configuration"`
	Model     string  `griffino:"key=DEEPSEEK_MODEL,type=options,name=Default Model,description=DeepSeek model used for chat requests.,optional=false,default=deepseek-chat,group=API Configuration,values=deepseek-chat:DeepSeek-V3 (Recommended)|deepseek-reasoner:DeepSeek-R1 (Reasoning)"`
	MaxTokens int     `griffino:"key=MAX_TOKENS,type=int,name=Max Tokens,description=Maximum completion tokens per request.,optional=true,default=4096,validation=min:256;max:8192,group=Model Parameters"`
	Temp      float64 `griffino:"key=TEMPERATURE,name=Temperature,description=Controls randomness.,optional=true,default=1.0,group=Model Parameters"`
	Verbose   bool    `griffino:"key=VERBOSE,name=Verbose,description=Verbose logging.,optional=true,default=true"`
	ignored   string  //nolint:unused
}

func TestParseConfigModel(t *testing.T) {
	def, err := ParseBootConfig(sampleBootConfig{})
	if err != nil {
		t.Fatalf("ParseBootConfig: %v", err)
	}
	if def.Kind != ConfigKindBoot {
		t.Fatalf("kind = %q, want boot", def.Kind)
	}
	if len(def.Fields) != 5 {
		t.Fatalf("got %d fields, want 5 (untagged/unexported skipped)", len(def.Fields))
	}

	apiKey := def.Fields[0]
	if apiKey.Key != "DEEPSEEK_API_KEY" || apiKey.Type != ConfigTypePassword {
		t.Errorf("apiKey = %+v", apiKey)
	}
	if apiKey.Name != "DeepSeek API Key" || apiKey.Description != "DeepSeek platform API key." {
		t.Errorf("apiKey name/desc = %q / %q", apiKey.Name, apiKey.Description)
	}
	if apiKey.Optional {
		t.Errorf("apiKey should be required")
	}
	if apiKey.HasDefault {
		t.Errorf("apiKey should have no default")
	}
	if apiKey.Group != "API Configuration" {
		t.Errorf("apiKey group = %q", apiKey.Group)
	}

	model := def.Fields[1]
	if model.Type != ConfigTypeOptions {
		t.Errorf("model type = %q", model.Type)
	}
	if !model.HasDefault || model.Default != "deepseek-chat" {
		t.Errorf("model default = %v (has=%t)", model.Default, model.HasDefault)
	}
	wantValues := []ConfigOption{
		{Value: "deepseek-chat", Display: "DeepSeek-V3 (Recommended)"},
		{Value: "deepseek-reasoner", Display: "DeepSeek-R1 (Reasoning)"},
	}
	if !reflect.DeepEqual(model.Values, wantValues) {
		t.Errorf("model values = %+v, want %+v", model.Values, wantValues)
	}

	maxTokens := def.Fields[2]
	if maxTokens.Type != ConfigTypeInt {
		t.Errorf("maxTokens type = %q", maxTokens.Type)
	}
	if !maxTokens.HasDefault || maxTokens.Default != 4096 {
		t.Errorf("maxTokens default = %v (%T)", maxTokens.Default, maxTokens.Default)
	}
	wantValidation := map[string]any{"minimum": 256, "maximum": 8192}
	if !reflect.DeepEqual(maxTokens.Validation, wantValidation) {
		t.Errorf("maxTokens validation = %+v, want %+v", maxTokens.Validation, wantValidation)
	}

	temp := def.Fields[3]
	if temp.Type != ConfigTypeFloat {
		t.Errorf("temp type = %q (inferred from float64)", temp.Type)
	}
	if !temp.HasDefault || temp.Default != 1.0 {
		t.Errorf("temp default = %v (%T)", temp.Default, temp.Default)
	}

	verbose := def.Fields[4]
	if verbose.Type != ConfigTypeBoolean {
		t.Errorf("verbose type = %q (inferred from bool)", verbose.Type)
	}
	if !verbose.HasDefault || verbose.Default != true {
		t.Errorf("verbose default = %v (%T)", verbose.Default, verbose.Default)
	}
}

func TestParseConfigModelErrors(t *testing.T) {
	if _, err := ParseConfigModel(nil, ConfigKindBoot); err == nil {
		t.Error("expected error for nil model")
	}
	if _, err := ParseConfigModel(42, ConfigKindBoot); err == nil {
		t.Error("expected error for non-struct model")
	}

	type missingKey struct {
		F string `griffino:"name=No Key,description=x"`
	}
	if _, err := ParseConfigModel(missingKey{}, ConfigKindBoot); err == nil {
		t.Error("expected error for missing key")
	}

	type badType struct {
		F []string `griffino:"key=K,name=N,description=D"`
	}
	if _, err := ParseConfigModel(badType{}, ConfigKindBoot); err == nil {
		t.Error("expected error for uninferrable type")
	}

	type badDefault struct {
		F int `griffino:"key=K,name=N,description=D,default=notint"`
	}
	if _, err := ParseConfigModel(badDefault{}, ConfigKindBoot); err == nil {
		t.Error("expected error for invalid int default")
	}
}

func TestParseUserConfigPointer(t *testing.T) {
	def, err := ParseUserConfig(&sampleBootConfig{})
	if err != nil {
		t.Fatalf("ParseUserConfig(pointer): %v", err)
	}
	if def.Kind != ConfigKindUser {
		t.Fatalf("kind = %q, want user", def.Kind)
	}
	if len(def.Fields) != 5 {
		t.Fatalf("got %d fields, want 5", len(def.Fields))
	}
}
