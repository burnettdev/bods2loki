package pipeline

import (
	"testing"
	"time"
)

func TestNewPipeline_Validation(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				APIKey:    "test-key",
				DatasetID: "7721",
				LineRefs:  []string{"1", "2"},
				LokiURL:   "http://localhost:3100",
				Interval:  30 * time.Second,
			},
			expectErr: false,
		},
		{
			name: "valid config with dry run",
			config: Config{
				APIKey:    "test-key",
				DatasetID: "7721",
				LineRefs:  []string{"1"},
				DryRun:    true,
				Interval:  30 * time.Second,
			},
			expectErr: false,
		},
		{
			name: "missing API key",
			config: Config{
				DatasetID: "7721",
				LineRefs:  []string{"1"},
				LokiURL:   "http://localhost:3100",
			},
			expectErr: true,
			errMsg:    "API key is required",
		},
		{
			name: "empty API key",
			config: Config{
				APIKey:    "",
				DatasetID: "7721",
				LineRefs:  []string{"1"},
				LokiURL:   "http://localhost:3100",
			},
			expectErr: true,
			errMsg:    "API key is required",
		},
		{
			name: "missing line refs",
			config: Config{
				APIKey:    "test-key",
				DatasetID: "7721",
				LineRefs:  []string{},
				LokiURL:   "http://localhost:3100",
			},
			expectErr: true,
			errMsg:    "at least one line reference is required",
		},
		{
			name: "nil line refs",
			config: Config{
				APIKey:    "test-key",
				DatasetID: "7721",
				LineRefs:  nil,
				LokiURL:   "http://localhost:3100",
			},
			expectErr: true,
			errMsg:    "at least one line reference is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pipeline, err := New(tt.config)

			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("Expected error %q, got %q", tt.errMsg, err.Error())
				}
				if pipeline != nil {
					t.Error("Expected nil pipeline on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if pipeline == nil {
					t.Error("Expected non-nil pipeline")
				}
			}
		})
	}
}

func TestNewPipeline_DryRunNoLokiClient(t *testing.T) {
	config := Config{
		APIKey:    "test-key",
		DatasetID: "7721",
		LineRefs:  []string{"1"},
		DryRun:    true,
		Interval:  30 * time.Second,
	}

	pipeline, err := New(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// In dry run mode, Loki client should be nil
	if pipeline.lokiClient != nil {
		t.Error("Expected lokiClient to be nil in dry run mode")
	}
}

func TestNewPipeline_ProductionHasLokiClient(t *testing.T) {
	config := Config{
		APIKey:    "test-key",
		DatasetID: "7721",
		LineRefs:  []string{"1"},
		LokiURL:   "http://localhost:3100",
		DryRun:    false,
		Interval:  30 * time.Second,
	}

	pipeline, err := New(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// In production mode, Loki client should be initialized
	if pipeline.lokiClient == nil {
		t.Error("Expected lokiClient to be initialized in production mode")
	}
}

func TestPipelineConfig_MultipleLineRefs(t *testing.T) {
	config := Config{
		APIKey:    "test-key",
		DatasetID: "7721",
		LineRefs:  []string{"1", "2", "3", "49x", "7"},
		LokiURL:   "http://localhost:3100",
		Interval:  30 * time.Second,
	}

	pipeline, err := New(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(pipeline.config.LineRefs) != 5 {
		t.Errorf("Expected 5 line refs, got %d", len(pipeline.config.LineRefs))
	}
}

func TestPipelineConfig_WithAuth(t *testing.T) {
	config := Config{
		APIKey:       "test-key",
		DatasetID:    "7721",
		LineRefs:     []string{"1"},
		LokiURL:      "https://logs.grafana.net",
		LokiUser:     "123456",
		LokiPassword: "glc_token",
		Interval:     30 * time.Second,
	}

	pipeline, err := New(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if pipeline.config.LokiUser != "123456" {
		t.Errorf("LokiUser = %q, want %q", pipeline.config.LokiUser, "123456")
	}
	if pipeline.config.LokiPassword != "glc_token" {
		t.Errorf("LokiPassword = %q, want %q", pipeline.config.LokiPassword, "glc_token")
	}
}

func TestConfig_Defaults(t *testing.T) {
	// Test that zero value Duration is handled
	config := Config{
		APIKey:   "test-key",
		LineRefs: []string{"1"},
	}

	// Pipeline should still be created even with zero interval
	pipeline, err := New(config)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Interval should be zero (caller is responsible for setting)
	if pipeline.config.Interval != 0 {
		t.Errorf("Expected 0 interval, got %v", pipeline.config.Interval)
	}
}
