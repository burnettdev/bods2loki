package parser

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// BusImageGenerator creates base64-encoded SVG images for bus visualization
type BusImageGenerator struct{}

func NewBusImageGenerator() *BusImageGenerator {
	return &BusImageGenerator{}
}

// GenerateBusImage creates a base64-encoded SVG image of a bus with line number and direction arrow
func (g *BusImageGenerator) GenerateBusImage(lineRef, direction string) string {
	// Determine arrow direction and color
	var arrow, color string
	switch strings.ToLower(direction) {
	case "inbound":
		arrow = "←"
		color = "#2E8B57" // Sea Green for inbound
	case "outbound":
		arrow = "→"
		color = "#FF6347" // Tomato Red for outbound
	default:
		arrow = "•"
		color = "#4682B4" // Steel Blue for unknown
	}

	// Create SVG with bus icon, line number, and directional arrow
	svg := fmt.Sprintf(`<svg width="120" height="60" xmlns="http://www.w3.org/2000/svg">
  <!-- Background -->
  <rect width="120" height="60" fill="#f8f9fa" stroke="#dee2e6" stroke-width="1" rx="8"/>
  
  <!-- Bus Body -->
  <rect x="15" y="20" width="50" height="25" fill="%s" rx="4"/>
  
  <!-- Bus Windows -->
  <rect x="18" y="23" width="8" height="6" fill="#87CEEB" rx="1"/>
  <rect x="28" y="23" width="8" height="6" fill="#87CEEB" rx="1"/>
  <rect x="38" y="23" width="8" height="6" fill="#87CEEB" rx="1"/>
  <rect x="48" y="23" width="8" height="6" fill="#87CEEB" rx="1"/>
  
  <!-- Bus Wheels -->
  <circle cx="25" cy="48" r="4" fill="#2F4F4F"/>
  <circle cx="55" cy="48" r="4" fill="#2F4F4F"/>
  <circle cx="25" cy="48" r="2" fill="#696969"/>
  <circle cx="55" cy="48" r="2" fill="#696969"/>
  
  <!-- Line Number -->
  <text x="75" y="25" font-family="Arial, sans-serif" font-size="14" font-weight="bold" fill="#333">%s</text>
  
  <!-- Direction Arrow -->
  <text x="75" y="45" font-family="Arial, sans-serif" font-size="20" font-weight="bold" fill="%s">%s</text>
</svg>`, color, lineRef, color, arrow)

	// Encode SVG to base64
	encoded := base64.StdEncoding.EncodeToString([]byte(svg))
	return fmt.Sprintf("data:image/svg+xml;base64,%s", encoded)
}

// getLineColor returns a unique color for each bus line
func (g *BusImageGenerator) getLineColor(lineRef string) string {
	// Color palette for different bus lines
	colors := map[string]string{
		"49x": "#E74C3C", // Red
		"7":   "#3498DB", // Blue
		"18":  "#2ECC71", // Green
		"42":  "#F39C12", // Orange
		"50":  "#9B59B6", // Purple
		"1":   "#1ABC9C", // Turquoise
		"2":   "#34495E", // Dark Blue
		"8":   "#E67E22", // Dark Orange
		"15":  "#8E44AD", // Dark Purple
		"20":  "#27AE60", // Dark Green
	}

	// Return specific color if found, otherwise generate based on line name
	if color, exists := colors[lineRef]; exists {
		return color
	}

	// Generate color based on line reference hash for consistency
	hash := 0
	for _, char := range lineRef {
		hash = int(char) + ((hash << 5) - hash)
	}

	// Convert to a pleasant color
	hue := (hash%360 + 360) % 360
	return fmt.Sprintf("hsl(%d, 70%%, 50%%)", hue)
}

// GenerateCompactBusImage creates a smaller, more compact bus image for dense displays
func (g *BusImageGenerator) GenerateCompactBusImage(lineRef, direction string) string {
	// Get line-specific color
	busColor := g.getLineColor(lineRef)

	// Direction indicator (using shapes instead of arrows)
	var directionShape, directionColor string
	switch strings.ToLower(direction) {
	case "inbound":
		directionShape = `<polygon points="45,22 50,25 45,28" fill="#28a745"/>` // Left-pointing triangle
		directionColor = "#28a745"
	case "outbound":
		directionShape = `<polygon points="50,22 55,25 50,28" fill="#dc3545"/>` // Right-pointing triangle
		directionColor = "#dc3545"
	default:
		directionShape = `<circle cx="50" cy="25" r="2" fill="#6c757d"/>` // Neutral dot
		directionColor = "#6c757d"
	}

	// Create enhanced compact SVG (90x45)
	svg := fmt.Sprintf(`<svg width="90" height="45" xmlns="http://www.w3.org/2000/svg">
  <!-- Background -->
  <rect width="90" height="45" fill="white" stroke="#dee2e6" stroke-width="1" rx="6"/>
  
  <!-- Bus Body (more detailed) -->
  <rect x="8" y="15" width="32" height="18" fill="%s" rx="3"/>
  
  <!-- Bus Front/Back -->
  <rect x="6" y="17" width="3" height="14" fill="%s" rx="1"/>
  
  <!-- Bus Windows (more realistic) -->
  <rect x="10" y="17" width="5" height="4" fill="#87CEEB" rx="1"/>
  <rect x="16" y="17" width="5" height="4" fill="#87CEEB" rx="1"/>
  <rect x="22" y="17" width="5" height="4" fill="#87CEEB" rx="1"/>
  <rect x="28" y="17" width="5" height="4" fill="#87CEEB" rx="1"/>
  <rect x="34" y="17" width="4" height="4" fill="#87CEEB" rx="1"/>
  
  <!-- Bus Door -->
  <rect x="18" y="22" width="8" height="9" fill="#2C3E50" rx="1"/>
  <rect x="19" y="23" width="6" height="7" fill="#34495E" rx="0.5"/>
  
  <!-- Wheels (more detailed) -->
  <circle cx="15" cy="35" r="3" fill="#2C3E50"/>
  <circle cx="31" cy="35" r="3" fill="#2C3E50"/>
  <circle cx="15" cy="35" r="1.5" fill="#7F8C8D"/>
  <circle cx="31" cy="35" r="1.5" fill="#7F8C8D"/>
  
  <!-- Line Number (larger, more prominent) -->
  <rect x="45" y="12" width="35" height="12" fill="%s" rx="2"/>
  <text x="62.5" y="21" font-family="Arial, sans-serif" font-size="9" font-weight="bold" fill="white" text-anchor="middle">%s</text>
  
  <!-- Direction Indicator -->
  %s
  
  <!-- Direction Label -->
  <text x="62.5" y="35" font-family="Arial, sans-serif" font-size="7" font-weight="bold" fill="%s" text-anchor="middle">%s</text>
</svg>`, busColor, busColor, busColor, lineRef, directionShape, directionColor, strings.ToUpper(direction[:2]))

	// Encode SVG to base64
	encoded := base64.StdEncoding.EncodeToString([]byte(svg))
	return fmt.Sprintf("data:image/svg+xml;base64,%s", encoded)
}

// GenerateStatusBadge creates a simple status badge image
func (g *BusImageGenerator) GenerateStatusBadge(lineRef, direction, status string) string {
	// Determine colors based on direction and status
	var bgColor, textColor string

	switch strings.ToLower(direction) {
	case "inbound":
		bgColor = "#198754" // Green
		textColor = "white"
	case "outbound":
		bgColor = "#dc3545" // Red
		textColor = "white"
	default:
		bgColor = "#6c757d" // Gray
		textColor = "white"
	}

	// Direction symbol
	arrow := "→"
	if strings.ToLower(direction) == "inbound" {
		arrow = "←"
	}

	// Create badge SVG
	svg := fmt.Sprintf(`<svg width="100" height="24" xmlns="http://www.w3.org/2000/svg">
  <!-- Badge Background -->
  <rect width="100" height="24" fill="%s" rx="12"/>
  
  <!-- Text Content -->
  <text x="50" y="16" font-family="Arial, sans-serif" font-size="11" font-weight="bold" 
        fill="%s" text-anchor="middle">%s %s %s</text>
</svg>`, bgColor, textColor, lineRef, arrow, strings.ToUpper(direction[:2]))

	// Encode SVG to base64
	encoded := base64.StdEncoding.EncodeToString([]byte(svg))
	return fmt.Sprintf("data:image/svg+xml;base64,%s", encoded)
}
