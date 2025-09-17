package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/example/paper2audio/internal/pdfx"
	"github.com/example/paper2audio/internal/script"
	"github.com/example/paper2audio/internal/summarize"
	"github.com/example/paper2audio/internal/tts"
	"gopkg.in/yaml.v3"
)

type Config struct {
	LLM struct {
		Provider    string `yaml:"provider"`     // openai|ollama|none
		Model       string `yaml:"model"`        // for openai
		OllamaModel string `yaml:"ollama_model"` // for ollama
	} `yaml:"llm"`
	TTS struct {
		Provider      string `yaml:"provider"` // piper|say|openai|elevenlabs
		PiperVoice    string `yaml:"piper_voice"`
		PiperVoiceJSON string `yaml:"piper_voice_json"`
		OpenAIVoice   string `yaml:"openai_voice"`
		ElevenVoice   string `yaml:"eleven_voice"`
	} `yaml:"tts"`
	Style struct {
		TargetMinutes int `yaml:"target_minutes"`
		WPM           int `yaml:"wpm"`
	} `yaml:"style"`
}

func loadConfig(path string) (*Config, error) {
	if path == "" {
		return &Config{}, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func main() {
	var (
		pdfPath    = flag.String("pdf", "", "Path to paper PDF")
		outAudio   = flag.String("out", "paper.m4a", "Output audio path (.m4a or .mp3)")
		outScript  = flag.String("script-out", "", "Optional path to save generated narration text")
		cfgPath    = flag.String("config", "", "Optional YAML config")
		llmProv    = flag.String("llm", "", "Override LLM provider (openai|ollama|none)")
		model      = flag.String("model", "", "Override LLM model (openai or ollama model name)")
		ttsProv    = flag.String("tts", "", "Override TTS provider (piper|say|openai|elevenlabs)")
		piperVoice = flag.String("piper-voice", "", "Path to Piper .onnx voice file")
		piperJSON  = flag.String("piper-json", "", "Path to Piper voice JSON metadata")
		openaiVoice= flag.String("openai-voice", "", "OpenAI TTS voice name")
		elevenVoice= flag.String("eleven-voice", "", "ElevenLabs voice name")
		minutes    = flag.Int("minutes", 12, "Target duration minutes (10–15 recommended)")
		wpm        = flag.Int("wpm", 150, "Words per minute for duration targeting")
	)
	flag.Parse()

	if *pdfPath == "" {
		log.Fatal("missing --pdf path")
	}

	cfg, err := loadConfig(*cfgPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	// CLI overrides
	if *llmProv != "" { cfg.LLM.Provider = *llmProv }
	if *model != "" {
		if cfg.LLM.Provider == "ollama" {
			cfg.LLM.OllamaModel = *model
		} else {
			cfg.LLM.Model = *model
		}
	}
	if *ttsProv != "" { cfg.TTS.Provider = *ttsProv }
	if *piperVoice != "" { cfg.TTS.PiperVoice = *piperVoice }
	if *piperJSON != ""  { cfg.TTS.PiperVoiceJSON = *piperJSON }
	if *openaiVoice != "" { cfg.TTS.OpenAIVoice = *openaiVoice }
	if *elevenVoice != "" { cfg.TTS.ElevenVoice = *elevenVoice }
	if *minutes > 0 { cfg.Style.TargetMinutes = *minutes }
	if *wpm > 0 { cfg.Style.WPM = *wpm }

	ctx := context.Background()

	// 1) Extract text & structure from PDF
	doc, err := pdfx.Extract(ctx, *pdfPath)
	if err != nil {
		log.Fatalf("pdf extract: %v", err)
	}
	// 2) Build outline + prompt and summarize into target word budget
	targetWords := script.TargetWordCount(cfg.Style.TargetMinutes, cfg.Style.WPM)
	prompt := script.MakePrompt(doc, targetWords)

	// 3) Summarize with selected backend
	var sum summarize.Summarizer
	switch cfg.LLM.Provider {
	case "openai":
		sum = summarize.NewOpenAI(cfg.LLM.Model)
	case "ollama":
		m := cfg.LLM.OllamaModel
		if m == "" { m = "llama3.1" }
		sum = summarize.NewOllama(m)
	case "none":
		sum = summarize.NewFallback()
	default:
		// default to fallback if not set
		sum = summarize.NewFallback()
	}
	narration, err := sum.Summarize(ctx, prompt)
	if err != nil {
		log.Fatalf("summarize: %v", err)
	}

	if *outScript != "" {
		if err := os.WriteFile(*outScript, []byte(narration), 0644); err != nil {
			log.Printf("warn: write script: %v", err)
		}
	}

	// 4) TTS with selected backend
	var speaker tts.Speaker
	switch cfg.TTS.Provider {
	case "piper":
		speaker = tts.NewPiper(cfg.TTS.PiperVoice, cfg.TTS.PiperVoiceJSON)
	case "say":
		speaker = tts.NewSay()
	case "openai":
		voice := cfg.TTS.OpenAIVoice
		if voice == "" { voice = "alloy" }
		speaker = tts.NewOpenAITTS(voice)
	case "elevenlabs":
		voice := cfg.TTS.ElevenVoice
		if voice == "" { voice = "Rachel" }
		speaker = tts.NewElevenLabs(voice)
	default:
		speaker = tts.NewSay() // mac quick default
	}

	if err := speaker.Speak(ctx, narration, *outAudio); err != nil {
		log.Fatalf("tts: %v", err)
	}

	fmt.Printf("✅ Wrote audio: %s\n", filepath.Clean(*outAudio))
}
