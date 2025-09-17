package script

import (
	"fmt"
	"math"
	"strings"

	"github.com/example/paper2audio/internal/pdfx"
)

func TargetWordCount(minutes, wpm int) int {
	if minutes <= 0 { minutes = 12 }
	if wpm <= 0 { wpm = 150 }
	w := minutes * wpm
	// keep a little headroom for pauses
	return int(math.Round(float64(w) * 0.9))
}

func MakePrompt(doc *pdfx.Doc, targetWords int) string {
	title := safe(doc.Title)
	authors := safe(doc.Authors)
	abstract := strings.TrimSpace(doc.Abstract)
	if len(abstract) > 1800 { abstract = abstract[:1800] }

	return fmt.Sprintf(`You are an expert technical writer and reviewer.
Produce a single-narrator audio-ready script of about %d words (±10%%) that presents a neutral, analytical overview of the following academic paper for a listener who is driving.

Hard requirements:
- Use concise, medium-length sentences with natural cadence.
- No hype, no back-and-forth dialog, no sycophancy.
- Include: (1) context & motivation, (2) precise problem statement, (3) core approach/method, (4) key findings/results with units if available, (5) strengths AND limitations/caveats, (6) bottom-line takeaway and when to read the full paper.
- Avoid formulas and long lists; prefer plain-language descriptions of what the method does.
- Cite section names sparingly in-line if clearly present (e.g., “In the evaluation section…”).
- Do not fabricate details not supported by the paper.

Paper metadata (may be noisy due to PDF extraction):
Title: %s
Authors: %s

Abstract (if detected):
%s

Longer extract (noisy):
%s

Now write the script only. Do not add headers or bullets.`, targetWords, title, authors, abstract, trimWords(doc.Body, 2500))
}

func trimWords(s string, n int) string {
	ws := strings.Fields(s)
	if len(ws) <= n { return s }
	return strings.Join(ws[:n], " ")
}

func safe(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
}
