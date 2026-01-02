// Package llm provides LLM integration for generating HMM scene prompts.
package llm

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	anthropicAPIURL = "https://api.anthropic.com/v1/messages"
	defaultModel    = "claude-sonnet-4-20250514"
)

// Client is an Anthropic API client.
type Client struct {
	apiKey     string
	httpClient *http.Client
	model      string
}

// SceneElements contains all the elements for generating an HMM scene.
type SceneElements struct {
	Character   string   // The Chinese character
	Pinyin      string   // Pronunciation
	Meaning     string   // Character meaning
	ActorName   string   // The actor (person)
	ActorDesc   string   // Actor description
	SetName     string   // The location
	SetDesc     string   // Set description
	ToneRoom    string   // The specific room/area
	ToneRoomDesc string  // Room description
	Props       []string // Props from components
	PropDescs   []string // Prop descriptions
}

// message represents an Anthropic API message.
type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// request represents an Anthropic API request.
type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	Messages  []message `json:"messages"`
}

// response represents an Anthropic API response.
type response struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// NewClient creates a new Anthropic client.
// It reads the API key from the ANTHROPIC_API_KEY environment variable.
func NewClient() (*Client, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	// Trim any whitespace/newlines that might have snuck in
	apiKey = strings.TrimSpace(apiKey)

	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		model: defaultModel,
	}, nil
}

// GenerateScene generates a vivid scene description for the given HMM elements.
func (c *Client) GenerateScene(elements SceneElements) (string, error) {
	prompt := buildPrompt(elements)

	req := request{
		Model:     c.model,
		MaxTokens: 300,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", c.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	var apiResp response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return "", fmt.Errorf("unmarshaling response: %w", err)
	}

	if apiResp.Error != nil {
		return "", fmt.Errorf("API error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Content) == 0 {
		return "", fmt.Errorf("empty response from API")
	}

	return strings.TrimSpace(apiResp.Content[0].Text), nil
}

// buildPrompt creates the prompt for the LLM.
func buildPrompt(e SceneElements) string {
	var sb strings.Builder

	sb.WriteString("You are helping create memorable mnemonic images for learning Chinese characters using the Hanzi Movie Method.\n\n")

	sb.WriteString("The system works like this:\n")
	sb.WriteString("- Each character becomes a vivid SCENE in a specific LOCATION\n")
	sb.WriteString("- An ACTOR (real or fictional person) performs an action\n")
	sb.WriteString("- PROPS represent the character's components\n")
	sb.WriteString("- The scene should be bizarre, emotional, and unforgettable\n\n")

	sb.WriteString("=== CHARACTER INFO ===\n")
	sb.WriteString(fmt.Sprintf("Character: %s\n", e.Character))
	sb.WriteString(fmt.Sprintf("Pronunciation: %s\n", e.Pinyin))
	if e.Meaning != "" {
		sb.WriteString(fmt.Sprintf("Meaning: %s\n", e.Meaning))
	}

	sb.WriteString("\n=== SCENE ELEMENTS ===\n")
	sb.WriteString(fmt.Sprintf("Actor: %s", e.ActorName))
	if e.ActorDesc != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", e.ActorDesc))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("Location: %s", e.SetName))
	if e.SetDesc != "" {
		sb.WriteString(fmt.Sprintf(" - %s", e.SetDesc))
	}
	sb.WriteString("\n")

	sb.WriteString(fmt.Sprintf("Specific area within location: %s", e.ToneRoom))
	if e.ToneRoomDesc != "" {
		sb.WriteString(fmt.Sprintf(" - %s", e.ToneRoomDesc))
	}
	sb.WriteString("\n")

	if len(e.Props) > 0 {
		sb.WriteString("\nProps (must appear in scene):\n")
		for i, prop := range e.Props {
			sb.WriteString(fmt.Sprintf("- %s", prop))
			if i < len(e.PropDescs) && e.PropDescs[i] != "" {
				sb.WriteString(fmt.Sprintf(": %s", e.PropDescs[i]))
			}
			sb.WriteString("\n")
		}
	}

	sb.WriteString("\n=== YOUR TASK ===\n")
	sb.WriteString("Generate an image prompt for an AI image generator (DALL-E, Midjourney, Stable Diffusion).\n\n")
	sb.WriteString("Requirements:\n")
	sb.WriteString("1. The actor must be clearly recognizable and doing something memorable\n")
	sb.WriteString("2. The location must be clearly the specified place and area\n")
	sb.WriteString("3. ALL props must be prominently featured and interacting with the actor\n")
	sb.WriteString("4. The scene should be slightly absurd or exaggerated to be memorable\n")
	sb.WriteString("5. Include visual style keywords at the end (e.g., 'digital art, cinematic lighting, detailed')\n\n")
	sb.WriteString("Output ONLY the image prompt, nothing else. Make it 2-4 sentences maximum.")

	return sb.String()
}
