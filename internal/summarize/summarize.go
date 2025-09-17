package summarize

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Summarizer interface {
	Summarize(ctx context.Context, prompt string) (string, error)
}

// -------- Fallback (very simple extractive) --------
type fallbackSummarizer struct{}

func NewFallback() Summarizer { return &fallbackSummarizer{} }

func (f *fallbackSummarizer) Summarize(ctx context.Context, prompt string) (string, error) {
	// Extremely simple: return the last block (the "script only" request)
	// In practice, you'd implement a basic heuristic extractive summary here.
	idx := strings.LastIndex(prompt, "Longer extract (noisy):")
	if idx < 0 {
		return strings.TrimSpace(prompt), nil
	}
	return strings.TrimSpace(prompt[:idx]), nil
}

// -------- OpenAI --------

type openAI struct{ Model string }

func NewOpenAI(model string) Summarizer { 
	if model == "" { model = "gpt-4o-mini" }
	return &openAI{Model: model} 
}

func (o *openAI) Summarize(ctx context.Context, prompt string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", errors.New("OPENAI_API_KEY not set")
	}
	body := map[string]any{
		"model": o.Model,
		"messages": []map[string]string{
			{"role":"system","content":"You write neutral, analytical, audio-ready summaries for technical papers."},
			{"role":"user","content": prompt},
		},
		"temperature": 0.2,
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{ Timeout: 60 * time.Second }
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		slurp, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai: %s", string(slurp))
	}
	var out struct {
		Choices []struct{
			Message struct{
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("openai: no choices")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), nil
}

// -------- Ollama (local) --------

type ollama struct{ Model string }

func NewOllama(model string) Summarizer {
	if model == "" { model = "llama3.1" }
	return &ollama{Model: model}
}

func (o *ollama) Summarize(ctx context.Context, prompt string) (string, error) {
	body := map[string]any{
		"model": o.Model,
		"prompt": prompt,
		"options": map[string]any{
			"temperature": 0.2,
		},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", "http://localhost:11434/api/generate", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{ Timeout: 120 * time.Second }
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		slurp, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama: %s", string(slurp))
	}

	// Streamed JSON lines: read and concat "response" fields
	var buf strings.Builder
	dec := json.NewDecoder(resp.Body)
	for dec.More() {
		var chunk map[string]any
		if err := dec.Decode(&chunk); err != nil { 
			if errors.Is(err, io.EOF) { break }
			return "", err 
		}
		if s, ok := chunk["response"].(string); ok {
			buf.WriteString(s)
		}
	}
	return strings.TrimSpace(buf.String()), nil
}
