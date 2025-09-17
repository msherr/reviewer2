package tts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

type Speaker interface {
	Speak(ctx context.Context, text, outPath string) error
}

// -------- Piper (local) --------
// Requires piper binary and a voice .onnx + .json
type piper struct{
	voice string
	voiceJSON string
}

func NewPiper(voicePath, voiceJSON string) Speaker {
	return &piper{voice: voicePath, voiceJSON: voiceJSON}
}

func (p *piper) Speak(ctx context.Context, text, outPath string) error {
	if p.voice == "" || p.voiceJSON == "" {
		return errors.New("piper: need --piper-voice and --piper-json")
	}
	tmp := outPath + ".tmp.txt"
	if err := os.WriteFile(tmp, []byte(text), 0644); err != nil {
		return err
	}
	defer os.Remove(tmp)

	ext := strings.ToLower(filepath.Ext(outPath))
	wavPath := outPath
	if ext != ".wav" {
		wavPath = outPath + ".wav"
	}
	cmd := exec.CommandContext(ctx, "piper", "-m", p.voice, "-c", p.voiceJSON, "-f", wavPath, "-q")
	cmd.Stdin, _ = os.Open(tmp)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("piper: %v: %s", err, string(out))
	}
	// Optionally convert wav→m4a if requested
	if ext == ".m4a" || ext == ".mp3" {
		if err := ffmpegConvert(ctx, wavPath, outPath); err != nil {
			return err
		}
		os.Remove(wavPath)
	}
	return nil
}

func ffmpegConvert(ctx context.Context, inWav, outFile string) error {
	// Requires ffmpeg installed
	cmd := exec.CommandContext(ctx, "ffmpeg", "-y", "-i", inWav, outFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg: %v: %s", err, string(out))
	}
	return nil
}

// -------- macOS say --------
type say struct{}

func NewSay() Speaker { return &say{} }

func (s *say) Speak(ctx context.Context, text, outPath string) error {
	// say can't write m4a directly; write to aiff then convert
	aiff := outPath + ".aiff"
	cmd := exec.CommandContext(ctx, "say", "-o", aiff, text)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("say: %v: %s", err, string(out))
	}
	ext := strings.ToLower(filepath.Ext(outPath))
	if ext == ".m4a" || ext == ".mp3" {
		if err := ffmpegConvert(ctx, aiff, outPath); err != nil {
			return err
		}
		os.Remove(aiff)
	} else {
		os.Rename(aiff, outPath)
	}
	return nil
}

// -------- OpenAI TTS --------
type openaiTTS struct{ Voice string }

func NewOpenAITTS(voice string) Speaker { return &openaiTTS{Voice: voice} }

func (o *openaiTTS) Speak(ctx context.Context, text, outPath string) error {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" { return errors.New("OPENAI_API_KEY not set") }
	// using audio/speech endpoint
	body := map[string]any{
		"model": "gpt-4o-mini-tts",
		"voice": o.Voice,
		"input": text,
		"format": "aac", // m4a container
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/speech", bytes.NewReader(b))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{ Timeout: 120 * time.Second }
	resp, err := client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		slurp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("openai tts: %s", string(slurp))
	}
	out, err := os.Create(outPath)
	if err != nil { return err }
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// -------- ElevenLabs --------
type eleven struct{ Voice string }

func NewElevenLabs(voice string) Speaker { return &eleven{Voice: voice} }

func (e *eleven) Speak(ctx context.Context, text, outPath string) error {
	apiKey := os.Getenv("ELEVEN_API_KEY")
	if apiKey == "" { return errors.New("ELEVEN_API_KEY not set") }
	url := fmt.Sprintf("https://api.elevenlabs.io/v1/text-to-speech/%s/stream", e.Voice)
	body := map[string]any{
		"text": text,
		"model_id": "eleven_multilingual_v2",
		"voice_settings": map[string]any{"stability":0.45,"similarity_boost":0.75},
	}
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(b))
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{ Timeout: 120 * time.Second }
	resp, err := client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		slurp, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("elevenlabs: %s", string(slurp))
	}
	out, err := os.Create(outPath)
	if err != nil { return err }
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}
