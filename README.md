# DISCLAIMER:  THIS SOFTWARE WAS ENTIRELY VIBE CODED.  I CHECKED NOTHING, AND MAKE ABSOLUTELY NO GUARANTEES ABOUT IT. 


# Reviewer2

Convert an academic paper **PDF → concise audio overview (~10–15 min)** for drive-time listening. 
Neutral tone, no hype, not a verbatim read-through.

## Features
- **Structured summary**: context → problem → approach → results → strengths/limits → takeaway
- **Duration targeting**: aims for ~10–15 min (≈130–180 wpm → 1.3k–2.7k words)
- **Pluggable LLM backends**: OpenAI (API), Ollama (local), or a simple fallback chunked extractive approach
- **Pluggable TTS backends**: Piper (local, recommended), macOS `say`, OpenAI TTS, ElevenLabs
- **Deterministic style**: neutral, analytical; keeps key caveats and limitations

## Install
```bash
# prerequisites (pick at least one LLM and one TTS backend)
# LLM options
# 1) OpenAI: set OPENAI_API_KEY
# 2) Ollama (local): install and run `ollama serve`

# TTS options
# a) Piper (local): https://github.com/rhasspy/piper
#    install a voice, e.g., en_US-amy-medium.onnx
# b) macOS 'say' (quickest to test)
# c) OpenAI TTS: set OPENAI_API_KEY and pick a voice
# d) ElevenLabs: set ELEVEN_API_KEY and pick a voice

# build
cd reviewer2
go mod tidy
go build ./cmd/reviewer2
```

## Usage
```bash
./reviewer2 \
  --pdf "/path/to/paper.pdf" \
  --out "/tmp/paper.m4a" \  --minutes 12 \  --llm openai \  --tts piper \  --piper-voice /path/to/en_US-amy-medium.onnx \  --piper-json  /path/to/en_US-amy-medium.onnx.json
```

Quick test with macOS TTS:
```bash
./reviewer2 --pdf paper.pdf --out paper.m4a --llm ollama --model llama3.1 --tts say
```

## Config (optional)
You can also use `config.yaml` instead of CLI flags:
```yaml
llm:
  provider: "openai"   # or "ollama" or "none"
  model: "gpt-4o-mini" # for openai
  ollama_model: "llama3.1"
tts:
  provider: "piper"    # piper|say|openai|elevenlabs
  piper_voice: "/voices/en_US-amy-medium.onnx"
  piper_voice_json: "/voices/en_US-amy-medium.onnx.json"
  openai_voice: "verse"
  eleven_voice: "Rachel"
style:
  target_minutes: 12
  wpm: 150
```

## What it Produces
- A single audio file (~10–15 min) with a neutral, drive-friendly narration containing:
  - why the paper exists (problem/motivation)
  - approach/method (not math derivations)
  - key findings/results (with unit/caveats)
  - strengths and clear limitations
  - bottom-line takeaway (is it worth your time to read deeply?)

## Notes on Accuracy and Hallucination
- The script prompts the LLM to **stick to the paper** and cite sections when possible.
- You can run entirely **offline** with Ollama + Piper.
- For critical use, skim the produced text (saved to `--script-out` if specified) before generating audio.

## License
MIT
