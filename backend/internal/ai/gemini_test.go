package ai

import "testing"

func TestDetectProviderKindAndModelCompatibility(t *testing.T) {
	cases := []struct {
		name     string
		cfg      AIConfig
		wantKind string
		wantLbl  string
	}{
		{
			name:     "Groq key auto-detected even if provider says Google Gemini",
			cfg:      AIConfig{Provider: "Google Gemini", GeminiAPIKey: "gsk_test1234567890", Model: "gemini-2.5-flash"},
			wantKind: "groq",
			wantLbl:  "Groq",
		},
		{
			name:     "OpenRouter key auto-detected",
			cfg:      AIConfig{Provider: "Google Gemini", GeminiAPIKey: "sk-or-v1-test1234"},
			wantKind: "openrouter",
			wantLbl:  "OpenRouter",
		},
		{
			name:     "Vertex Express key auto-detected",
			cfg:      AIConfig{GeminiAPIKey: "AQ.Ab8test1234"},
			wantKind: "vertex",
			wantLbl:  "Vertex AI Express",
		},
		{
			name:     "Google AI Studio key auto-detected",
			cfg:      AIConfig{GeminiAPIKey: "AIzaSyTest1234"},
			wantKind: "gemini",
			wantLbl:  "Google AI Studio",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, lbl, _, defaults := detectProviderKind(tc.cfg)
			if kind != tc.wantKind || lbl != tc.wantLbl || len(defaults) == 0 {
				t.Fatalf("got (%s, %s, %v), want (%s, %s)", kind, lbl, defaults, tc.wantKind, tc.wantLbl)
			}
		})
	}

	if isModelCompatibleWithProvider("groq", "gemini-2.5-flash") {
		t.Fatalf("gemini-2.5-flash should NOT be compatible with groq")
	}
	if !isModelCompatibleWithProvider("groq", "llama-3.3-70b-versatile") {
		t.Fatalf("llama-3.3-70b-versatile MUST be compatible with groq")
	}

	raw := "<think>planning...</think>\n```json\n{\"status\":\"ok\"}\n```"
	if got := cleanAIJSONResponse(raw); got != `{"status":"ok"}` {
		t.Fatalf("unexpected cleaned JSON: %q", got)
	}
}
