package parser

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestGenerateCompactBusImage(t *testing.T) {
	generator := NewBusImageGenerator()

	tests := []struct {
		name           string
		lineRef        string
		direction      string
		expectContains []string // These are checked in the decoded SVG content
	}{
		{
			name:      "inbound direction",
			lineRef:   "1",
			direction: "inbound",
			expectContains: []string{
				"<svg",
				"#28a745", // Green color for inbound
			},
		},
		{
			name:      "outbound direction",
			lineRef:   "2",
			direction: "outbound",
			expectContains: []string{
				"<svg",
				"#dc3545", // Red color for outbound
			},
		},
		{
			name:      "unknown direction",
			lineRef:   "3",
			direction: "unknown",
			expectContains: []string{
				"<svg",
				"#6c757d", // Gray color for unknown
			},
		},
		{
			name:      "case insensitive direction",
			lineRef:   "1",
			direction: "INBOUND",
			expectContains: []string{
				"#28a745", // Green color for inbound
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.GenerateCompactBusImage(tt.lineRef, tt.direction)

			// Verify it's base64 encoded data URI
			if !strings.HasPrefix(result, "data:image/svg+xml;base64,") {
				t.Error("Result should start with data:image/svg+xml;base64,")
			}

			// Decode base64 and check SVG content
			b64Data := strings.TrimPrefix(result, "data:image/svg+xml;base64,")
			decoded, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				t.Fatalf("Failed to decode base64: %v", err)
			}

			svgContent := string(decoded)
			for _, expected := range tt.expectContains {
				if !strings.Contains(svgContent, expected) {
					t.Errorf("SVG should contain %q", expected)
				}
			}

			// Verify line number appears in SVG
			if !strings.Contains(svgContent, tt.lineRef) {
				t.Errorf("SVG should contain line reference %q", tt.lineRef)
			}
		})
	}
}

func TestGenerateBusImage(t *testing.T) {
	generator := NewBusImageGenerator()

	tests := []struct {
		name           string
		lineRef        string
		direction      string
		expectContains []string
	}{
		{
			name:      "inbound with left arrow",
			lineRef:   "49x",
			direction: "inbound",
			expectContains: []string{
				"#2E8B57", // Sea Green for inbound
				"←",       // Left arrow
			},
		},
		{
			name:      "outbound with right arrow",
			lineRef:   "7",
			direction: "outbound",
			expectContains: []string{
				"#FF6347", // Tomato Red for outbound
				"→",       // Right arrow
			},
		},
		{
			name:      "unknown direction with dot",
			lineRef:   "X",
			direction: "",
			expectContains: []string{
				"#4682B4", // Steel Blue for unknown
				"•",       // Dot
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.GenerateBusImage(tt.lineRef, tt.direction)

			// Decode and check SVG
			b64Data := strings.TrimPrefix(result, "data:image/svg+xml;base64,")
			decoded, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				t.Fatalf("Failed to decode base64: %v", err)
			}

			svgContent := string(decoded)
			for _, expected := range tt.expectContains {
				if !strings.Contains(svgContent, expected) {
					t.Errorf("SVG should contain %q", expected)
				}
			}
		})
	}
}

func TestGetLineColor(t *testing.T) {
	generator := NewBusImageGenerator()

	tests := []struct {
		name        string
		lineRef     string
		expectColor string
	}{
		{"predefined 49x", "49x", "#E74C3C"},
		{"predefined 7", "7", "#3498DB"},
		{"predefined 1", "1", "#1ABC9C"},
		{"predefined 2", "2", "#34495E"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			color := generator.getLineColor(tt.lineRef)
			if color != tt.expectColor {
				t.Errorf("getLineColor(%q) = %q, want %q", tt.lineRef, color, tt.expectColor)
			}
		})
	}
}

func TestGetLineColor_HashBased(t *testing.T) {
	generator := NewBusImageGenerator()

	// Lines not in predefined list should get hash-based colors
	unknownLines := []string{"99", "100A", "EXPRESS", "X1"}

	for _, lineRef := range unknownLines {
		t.Run(lineRef, func(t *testing.T) {
			color := generator.getLineColor(lineRef)

			// Should return HSL color format
			if !strings.HasPrefix(color, "hsl(") {
				t.Errorf("getLineColor(%q) = %q, expected HSL format", lineRef, color)
			}

			// Same input should always give same output (deterministic)
			color2 := generator.getLineColor(lineRef)
			if color != color2 {
				t.Errorf("getLineColor should be deterministic: got %q and %q", color, color2)
			}
		})
	}
}

func TestGenerateStatusBadge(t *testing.T) {
	generator := NewBusImageGenerator()

	tests := []struct {
		name           string
		lineRef        string
		direction      string
		status         string
		expectContains []string
	}{
		{
			name:      "inbound badge",
			lineRef:   "1",
			direction: "inbound",
			status:    "running",
			expectContains: []string{
				"#198754", // Green for inbound
				"←",       // Inbound arrow
				"IN",      // Direction abbreviation
			},
		},
		{
			name:      "outbound badge",
			lineRef:   "2",
			direction: "outbound",
			status:    "running",
			expectContains: []string{
				"#dc3545", // Red for outbound
				"→",       // Outbound arrow
				"OU",      // Direction abbreviation
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generator.GenerateStatusBadge(tt.lineRef, tt.direction, tt.status)

			b64Data := strings.TrimPrefix(result, "data:image/svg+xml;base64,")
			decoded, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				t.Fatalf("Failed to decode base64: %v", err)
			}

			svgContent := string(decoded)
			for _, expected := range tt.expectContains {
				if !strings.Contains(svgContent, expected) {
					t.Errorf("Badge SVG should contain %q", expected)
				}
			}
		})
	}
}

func TestImageGeneratorDeterministic(t *testing.T) {
	generator := NewBusImageGenerator()

	// Generate same image multiple times
	images := make([]string, 3)
	for i := range images {
		images[i] = generator.GenerateCompactBusImage("49x", "inbound")
	}

	// All should be identical
	for i := 1; i < len(images); i++ {
		if images[i] != images[0] {
			t.Errorf("Image generation should be deterministic, got different results")
		}
	}
}

func TestValidBase64Output(t *testing.T) {
	generator := NewBusImageGenerator()

	// Test all generator methods produce valid base64
	generators := []struct {
		name   string
		result string
	}{
		{"GenerateBusImage", generator.GenerateBusImage("1", "inbound")},
		{"GenerateCompactBusImage", generator.GenerateCompactBusImage("1", "inbound")},
		{"GenerateStatusBadge", generator.GenerateStatusBadge("1", "inbound", "active")},
	}

	for _, g := range generators {
		t.Run(g.name, func(t *testing.T) {
			if !strings.HasPrefix(g.result, "data:image/svg+xml;base64,") {
				t.Errorf("%s should return data URI format", g.name)
			}

			b64Data := strings.TrimPrefix(g.result, "data:image/svg+xml;base64,")
			decoded, err := base64.StdEncoding.DecodeString(b64Data)
			if err != nil {
				t.Errorf("%s produced invalid base64: %v", g.name, err)
			}

			if !strings.Contains(string(decoded), "<svg") {
				t.Errorf("%s should contain SVG markup", g.name)
			}
		})
	}
}
