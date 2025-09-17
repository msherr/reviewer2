package pdfx

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	pdf "github.com/ledongthuc/pdf"
)

type Doc struct {
	Title    string
	Authors  string
	Abstract string
	Headings []string
	Body     string
}

func Extract(ctx context.Context, path string) (*Doc, error) {
	f, r, err := pdf.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var buf bytes.Buffer
	b, err := r.GetPlainText()
	if err != nil {
		return nil, err
	}
	if _, err := io.Copy(&buf, b); err != nil {
		return nil, err
	}
	raw := buf.String()
	// naive heuristics for academic PDFs
	title, authors, abstract := sniffFrontMatter(raw)
	headings := sniffHeadings(raw)

	return &Doc{
		Title:    title,
		Authors:  authors,
		Abstract: abstract,
		Headings: headings,
		Body:     raw,
	}, nil
}

// Simple heuristics (you can refine later)
func sniffFrontMatter(text string) (title, authors, abstract string) {
	lines := strings.Split(text, "\n")
	trimmed := make([]string, 0, len(lines))
	for _, l := range lines {
		l2 := strings.TrimSpace(l)
		if l2 != "" {
			trimmed = append(trimmed, l2)
		}
	}
	if len(trimmed) > 0 {
		title = trimmed[0]
	}
	if len(trimmed) > 1 {
		authors = trimmed[1]
	}
	// Abstract
	abstract = findAbstract(text)
	return
}

func findAbstract(text string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, "abstract")
	if idx < 0 {
		return ""
	}
	// grab ~250–400 words after "abstract"
	after := text[idx:]
	words := strings.Fields(after)
	if len(words) > 400 { words = words[:400] }
	return strings.Join(words, " ")
}

func sniffHeadings(text string) []string {
	var hs []string
	for _, h := range []string{"introduction", "background", "related work", "method", "methods", "approach", "evaluation", "results", "discussion", "limitations", "conclusion", "future work"} {
		if strings.Contains(strings.ToLower(text), h) {
			hs = append(hs, h)
		}
	}
	return hs
}

// Utility for debugging
func SaveDebug(path string, d *Doc) error {
	out := fmt.Sprintf("Title: %s\nAuthors: %s\nAbstract: %s\nHeadings: %v\n", d.Title, d.Authors, d.Abstract, d.Headings)
	return os.WriteFile(path, []byte(out), 0644)
}
