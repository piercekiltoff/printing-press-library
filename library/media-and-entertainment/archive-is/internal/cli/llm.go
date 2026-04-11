// LLM provider detection and invocation for the tldr subcommand.
//
// Provider selection order (first hit wins):
//  1. `claude` binary on PATH — Matt has Claude Code installed; shelling out
//     is the lowest-friction path
//  2. ANTHROPIC_API_KEY env var — direct call to api.anthropic.com/v1/messages
//     with claude-haiku-4-5 (fast, cheap, sufficient for summaries)
//  3. OPENAI_API_KEY env var — direct call to api.openai.com/v1/chat/completions
//     with gpt-4o-mini
//  4. None → graceful error with install hints
//
// The prompt is intentionally constrained: "3 bullets and 1 headline." The
// goal is a tl;dr, not an essay.

package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// summaryPrompt is the instruction prefix passed to the LLM. Tuned for the
// terminal-reading use case: concise, scannable, no fluff.
const summaryPrompt = `Summarize the following article. Output format:

HEADLINE: <one-line headline capturing the main point>

BULLETS:
- <bullet 1 — key finding or event>
- <bullet 2 — supporting detail or context>
- <bullet 3 — implication, quote, or consequence>

Keep bullets under 25 words each. No preamble, no meta-commentary. Start with the HEADLINE line.

Article text:

`

// maxArticleChars caps article text sent to the LLM. Both claude-haiku-4-5
// and gpt-4o-mini handle more, but 24k chars (~6k tokens) is plenty for a
// tl;dr and keeps latency + cost predictable.
const maxArticleChars = 24000

// Summary is the structured output returned by llmSummarize. The headline and
// bullets are extracted from the LLM's free-form response.
type Summary struct {
	Headline string   `json:"headline"`
	Bullets  []string `json:"bullets"`
	Provider string   `json:"provider"`
	Raw      string   `json:"raw,omitempty"`
}

// llmSummarize sends article text to the first available LLM provider and
// returns a parsed Summary. Returns an error listing all three install options
// if no provider is found.
func llmSummarize(text string) (*Summary, error) {
	if len(text) == 0 {
		return nil, fmt.Errorf("no article text to summarize")
	}
	truncated := false
	if len(text) > maxArticleChars {
		text = text[:maxArticleChars]
		truncated = true
	}
	prompt := summaryPrompt + text
	if truncated {
		prompt += "\n\n[Article truncated for length.]"
	}

	// 1. Try Claude Code CLI first.
	if _, err := exec.LookPath("claude"); err == nil {
		raw, err := summarizeViaClaudeCLI(prompt)
		if err == nil {
			s := parseSummary(raw)
			s.Provider = "claude-cli"
			return s, nil
		}
		// Fall through to API-based providers if Claude CLI fails.
	}

	// 2. Try Anthropic API.
	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		raw, err := summarizeViaAnthropic(prompt, key)
		if err != nil {
			return nil, fmt.Errorf("anthropic API: %w", err)
		}
		s := parseSummary(raw)
		s.Provider = "anthropic-api"
		return s, nil
	}

	// 3. Try OpenAI API.
	if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		raw, err := summarizeViaOpenAI(prompt, key)
		if err != nil {
			return nil, fmt.Errorf("openai API: %w", err)
		}
		s := parseSummary(raw)
		s.Provider = "openai-api"
		return s, nil
	}

	return nil, fmt.Errorf(`tl;dr requires an LLM provider. Options:
  1. Install Claude Code:   https://claude.com/claude-code
  2. Set ANTHROPIC_API_KEY:  export ANTHROPIC_API_KEY=sk-ant-...
  3. Set OPENAI_API_KEY:    export OPENAI_API_KEY=sk-...`)
}

// summarizeViaClaudeCLI shells out to `claude --print <prompt>`. The --print
// flag makes claude CLI operate in non-interactive one-shot mode suitable
// for pipelines.
func summarizeViaClaudeCLI(prompt string) (string, error) {
	cmd := exec.Command("claude", "--print", prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude cli: %w (stderr: %s)", err, stderr.String())
	}
	return stdout.String(), nil
}

// summarizeViaAnthropic calls api.anthropic.com/v1/messages directly.
// Uses claude-haiku-4-5 which is fast, cheap, and more than sufficient for
// summarization. The model name is pinned for stability.
func summarizeViaAnthropic(prompt, apiKey string) (string, error) {
	reqBody := map[string]any{
		"model":      "claude-haiku-4-5",
		"max_tokens": 500,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("content-type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	var out strings.Builder
	for _, c := range result.Content {
		if c.Type == "text" {
			out.WriteString(c.Text)
		}
	}
	return out.String(), nil
}

// summarizeViaOpenAI calls api.openai.com/v1/chat/completions directly.
// Uses gpt-4o-mini which is the cost/quality sweet spot for this task.
func summarizeViaOpenAI(prompt, apiKey string) (string, error) {
	reqBody := map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": 500,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}
	return result.Choices[0].Message.Content, nil
}

// parseSummary extracts the headline and bullets from a free-form LLM response.
// The prompt asks for a specific format (HEADLINE: line, then BULLETS: list),
// but LLMs are free-form so we parse defensively.
func parseSummary(raw string) *Summary {
	s := &Summary{Raw: strings.TrimSpace(raw)}
	lines := strings.Split(raw, "\n")

	inBullets := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Headline detection: prefix match on "HEADLINE:" (case-insensitive).
		if strings.HasPrefix(strings.ToUpper(trimmed), "HEADLINE:") {
			s.Headline = strings.TrimSpace(trimmed[len("HEADLINE:"):])
			continue
		}
		// Bullet section start.
		if strings.HasPrefix(strings.ToUpper(trimmed), "BULLETS:") {
			inBullets = true
			continue
		}
		// Bullet line — various bullet styles.
		if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") || strings.HasPrefix(trimmed, "• ") {
			inBullets = true
			bullet := strings.TrimSpace(trimmed[1:])
			if bullet != "" {
				s.Bullets = append(s.Bullets, bullet)
			}
			continue
		}
		// If we're in the bullets section and the line looks like a continuation, append.
		if inBullets && s.Headline != "" {
			// Numbered lists
			if len(trimmed) > 2 && trimmed[0] >= '0' && trimmed[0] <= '9' && (trimmed[1] == '.' || trimmed[1] == ')') {
				bullet := strings.TrimSpace(trimmed[2:])
				if bullet != "" {
					s.Bullets = append(s.Bullets, bullet)
				}
			}
		}
	}

	// Fallback: if parsing produced nothing structured, use the whole response as a single bullet.
	if s.Headline == "" && len(s.Bullets) == 0 && s.Raw != "" {
		s.Headline = "(unparsed)"
		s.Bullets = []string{s.Raw}
	}

	return s
}
