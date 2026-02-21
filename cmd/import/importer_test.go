package importcmd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/FreeDictionary/load2db-go/db"
)

func TestNormalizeJSON(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantWord     string
		wantLang     string
		wantLangCode string
		wantErr      bool
	}{
		{
			name:         "valid minimal entry",
			input:        `{"word":"hello","lang":"English","lang_code":"en"}`,
			wantWord:     "hello",
			wantLang:     "English",
			wantLangCode: "en",
			wantErr:      false,
		},
		{
			name:         "valid entry with extra fields",
			input:        `{"word":"test","lang":"Test Language","lang_code":"test","pos":"noun","senses":[{"glosses":["a test"]}]}`,
			wantWord:     "test",
			wantLang:     "Test Language",
			wantLangCode: "test",
			wantErr:      false,
		},
		{
			name:         "missing word",
			input:        `{"lang":"English","lang_code":"en"}`,
			wantWord:     "",
			wantLang:     "English",
			wantLangCode: "en",
			wantErr:      true,
		},
		{
			name:         "missing lang_code",
			input:        `{"word":"hello","lang":"English"}`,
			wantWord:     "hello",
			wantLang:     "English",
			wantLangCode: "",
			wantErr:      true,
		},
		{
			name:         "invalid JSON",
			input:        `{"word":"hello",invalid}`,
			wantWord:     "",
			wantLang:     "",
			wantLangCode: "",
			wantErr:      true,
		},
		{
			name:         "empty word",
			input:        `{"word":"","lang":"English","lang_code":"en"}`,
			wantWord:     "",
			wantLang:     "English",
			wantLangCode: "en",
			wantErr:      true,
		},
		{
			name:         "JSON normalization - key order",
			input:        `{"lang_code":"en","word":"hello","lang":"English"}`,
			wantWord:     "hello",
			wantLang:     "English",
			wantLangCode: "en",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importer := NewImporter(ImporterConfig{
				DB:           nil,
				BatchSize:    1000,
				ShowProgress: false,
			})

			got, err := importer.normalizeJSON([]byte(tt.input))

			if (err != nil) != tt.wantErr {
				t.Errorf("normalizeJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.Word != tt.wantWord {
					t.Errorf("normalizeJSON().Word = %v, want %v", got.Word, tt.wantWord)
				}
				if got.Lang != tt.wantLang {
					t.Errorf("normalizeJSON().Lang = %v, want %v", got.Lang, tt.wantLang)
				}
				if got.LangCode != tt.wantLangCode {
					t.Errorf("normalizeJSON().LangCode = %v, want %v", got.LangCode, tt.wantLangCode)
				}

				// Verify the data is valid JSON
				var rawData map[string]interface{}
				if err := json.Unmarshal(got.Data, &rawData); err != nil {
					t.Errorf("normalizeJSON().Data is not valid JSON: %v", err)
				}
			}
		})
	}
}

func TestNormalizeJSONPreservesData(t *testing.T) {
	importer := NewImporter(ImporterConfig{
		DB:           nil,
		BatchSize:    1000,
		ShowProgress: false,
	})

	// Complex Wiktionary entry
	input := `{
		"word": "example",
		"lang": "English",
		"lang_code": "en",
		"pos": "noun",
		"senses": [
			{
				"glosses": ["A sample, model, or instance"],
				"examples": [
					{"text": "This is an example sentence"}
				]
			}
		],
		"translations": [
			{
				"lang_code": "es",
				"translation": "ejemplo"
			}
		]
	}`

	got, err := importer.normalizeJSON([]byte(input))
	if err != nil {
		t.Fatalf("normalizeJSON() unexpected error: %v", err)
	}

	// Parse the normalized data back
	var normalized map[string]interface{}
	if err := json.Unmarshal(got.Data, &normalized); err != nil {
		t.Fatalf("Failed to unmarshal normalized data: %v", err)
	}

	// Verify structure is preserved
	if normalized["word"] != "example" {
		t.Errorf("word field not preserved correctly")
	}
	if normalized["lang"] != "English" {
		t.Errorf("lang field not preserved correctly")
	}
	if normalized["lang_code"] != "en" {
		t.Errorf("lang_code field not preserved correctly")
	}
	if normalized["pos"] != "noun" {
		t.Errorf("pos field not preserved correctly")
	}

	// Verify nested structures
	senses, ok := normalized["senses"].([]interface{})
	if !ok || len(senses) == 0 {
		t.Errorf("senses array not preserved correctly")
	}

	translations, ok := normalized["translations"].([]interface{})
	if !ok || len(translations) == 0 {
		t.Errorf("translations array not preserved correctly")
	}
}

func TestWordDataToWordEntry(t *testing.T) {
	// This test verifies the WordData struct can parse Wiktionary JSON
	input := `{
		"word": "test",
		"lang": "English",
		"lang_code": "en",
		"pos": "noun"
	}`

	var data WordData
	if err := json.Unmarshal([]byte(input), &data); err != nil {
		t.Fatalf("Failed to unmarshal WordData: %v", err)
	}

	if data.Word != "test" {
		t.Errorf("Word = %v, want %v", data.Word, "test")
	}
	if data.Lang != "English" {
		t.Errorf("Lang = %v, want %v", data.Lang, "English")
	}
	if data.LangCode != "en" {
		t.Errorf("LangCode = %v, want %v", data.LangCode, "en")
	}
	if data.Pos != "noun" {
		t.Errorf("Pos = %v, want %v", data.Pos, "noun")
	}
}

func TestBatchInsertWords(t *testing.T) {
	// This is a placeholder for integration tests
	// In a real scenario, you would need a test PostgreSQL instance
	ctx := context.Background()
	importer := NewImporter(ImporterConfig{
		DB:           nil,
		BatchSize:    1000,
		ShowProgress: false,
	})

	// Test empty batch
	err := importer.DB.BatchInsertWords(ctx, []db.WordEntry{})
	if err != nil {
		t.Errorf("BatchInsertWords with empty batch failed: %v", err)
	}

	// Note: Full integration test requires a running PostgreSQL instance
	// Use testcontainers or a dedicated test database for full testing
	t.Log("Note: Full batch insert testing requires a PostgreSQL instance")
}
