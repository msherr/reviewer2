package script

import (
	"fmt"
	"math"
	"strings"

	"github.com/msherr/reviewer2/internal/pdfx"
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
Produce a single-narrator audio-ready script of about %d words (±10%%) that presents a rigorous, technically detailed overview of the following academic paper for a listener who is a subject-matter expert.

Hard requirements:
- Use concise, medium-length sentences with natural cadence suitable for audio.
- Maintain a neutral, analytical tone; no hype, dialog, or sycophancy.
- Cover in order: (1) research context and explicit motivation, (2) formal problem statement and assumptions, (3) technical approach with architecture/algorithm specifics, training or implementation details, and key hyperparameters, (4) evaluation setup including datasets, baselines, metrics, and quantitative results, (5) critical analysis of strengths, limitations, and failure cases, (6) actionable takeaway highlighting when the method is useful and when the full paper merits deeper study.
- Emphasize concrete technical details that informed peers would expect; mention ablations, resource requirements, or theoretical guarantees if the paper provides them.
- Cite section names or figure references sparingly when they provide orientation (e.g., “In Section 4’s evaluation…”).
- Do not fabricate details; if information is missing in the paper, state that explicitly instead of guessing.

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
