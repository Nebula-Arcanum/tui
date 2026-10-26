package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ZenNotes/tui/internal/config"
)

const tikzFence = "```tikz"

// useMath points the config dir at a fresh temp dir whose config.toml sets
// [terminal] math to engine ("" writes no config.toml at all).
func useMath(t *testing.T, engine string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("ZENNOTES_CONFIG_DIR", dir)
	if engine != "" {
		writeMath(t, dir, engine)
	}
	return dir
}

func writeMath(t *testing.T, dir, engine string) {
	t.Helper()
	body := "[terminal]\nmath = \"" + engine + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

// katexParagraph is the "Math → KaTeX" paragraph of DefaultInstructions,
// up to the blank line that ends it.
func katexParagraph(t *testing.T) string {
	t.Helper()
	start := strings.Index(DefaultInstructions, "**Math → KaTeX.**")
	if start < 0 {
		t.Fatal("DefaultInstructions has no \"Math → KaTeX\" paragraph")
	}
	end := strings.Index(DefaultInstructions[start:], "\n\n")
	if end < 0 {
		t.Fatal("the KaTeX paragraph has no end")
	}
	return DefaultInstructions[start : start+end+2]
}

func TestInstructionsLatexAsksForBareTikz(t *testing.T) { // AC64
	useMath(t, "latex")
	got := ResolveInstructions()
	if !strings.Contains(got, `\begin{tikzpicture}`) {
		t.Fatalf("AC64: latex instructions do not ask for a \\begin{tikzpicture} block")
	}
	if n := strings.Count(got, tikzFence); n != 0 {
		t.Fatalf("AC64: latex instructions still ask for %s fences (%d mentions)", tikzFence, n)
	}
	if strings.Contains(got, `\usetikzlibrary{arrows.meta,calc}`) {
		t.Fatalf("AC64: the latex scaffold still loads its own library with \\usetikzlibrary")
	}
	if !strings.Contains(got, "preamble.tex") {
		t.Fatalf("AC64: latex instructions do not say libraries and packages come from preamble.tex")
	}
	if !strings.Contains(got, "arrows.meta,calc") {
		t.Fatalf("AC64: latex instructions do not say the default preamble loads arrows.meta,calc")
	}
	// The scaffold is a bare block: the line above its \begin{tikzpicture}
	// is not a code fence of any language.
	lines := strings.Split(got, "\n")
	found := false
	for i, l := range lines {
		if !strings.HasPrefix(l, `\begin{tikzpicture}[>=Stealth`) {
			continue
		}
		found = true
		if i > 0 && strings.HasPrefix(strings.TrimSpace(lines[i-1]), "```") {
			t.Fatalf("AC64: the TikZ scaffold is still fenced: %q", lines[i-1])
		}
	}
	if !found {
		t.Fatalf("AC64: latex instructions lost the TikZ scaffold")
	}
}

func TestInstructionsLatexKeepsKatexParagraph(t *testing.T) { // AC65
	useMath(t, "latex")
	if want := katexParagraph(t); !strings.Contains(ResolveInstructions(), want) {
		t.Fatalf("AC65: the Math → KaTeX paragraph changed under latex; want byte-equal:\n%s", want)
	}
}

func TestInstructionsTypstIsDefault(t *testing.T) { // AC66
	for _, engine := range []string{"typst", ""} {
		useMath(t, engine)
		if got := ResolveInstructions(); got != DefaultInstructions {
			t.Fatalf("AC66: math=%q instructions differ from DefaultInstructions", engine)
		}
	}
}

func TestInstructionsOverrideWinsForBothEngines(t *testing.T) { // AC67
	const override = "My own rules.\n```tikz\nkeep me\n```\n"
	for _, engine := range []string{"typst", "latex"} {
		useMath(t, engine)
		if err := os.WriteFile(config.MCPInstructionsPath(), []byte(override), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := ResolveInstructions(); got != override {
			t.Fatalf("AC67: math=%s served %q, want the override unchanged", engine, got)
		}
	}
}

// Risk guard: latexInstructions is DefaultInstructions through exact-substring
// replacements. If DefaultInstructions is edited so a replacement no longer
// matches, a fence or the scaffold's library line survives into latex.
func TestLatexReplacementsAllApply(t *testing.T) {
	if !strings.Contains(DefaultInstructions, tikzFence) ||
		!strings.Contains(DefaultInstructions, `\usetikzlibrary{arrows.meta,calc}`) {
		t.Fatal("DefaultInstructions no longer carries the tikz fences this guard checks")
	}
	for i := 0; i < len(latexReplacements); i += 2 {
		if n := strings.Count(DefaultInstructions, latexReplacements[i]); n != 1 {
			t.Errorf("latex replacement %d matches DefaultInstructions %d times, want 1: %q", i/2, n, latexReplacements[i])
		}
	}
	if latexInstructions == DefaultInstructions {
		t.Fatal("latexInstructions is DefaultInstructions: no replacement applied")
	}
	if n := strings.Count(latexInstructions, tikzFence); n != 0 {
		t.Fatalf("%d %s mention(s) survived into latexInstructions", n, tikzFence)
	}
	if strings.Contains(latexInstructions, `\usetikzlibrary{arrows.meta,calc}`) {
		t.Fatal("the scaffold's \\usetikzlibrary line survived into latexInstructions")
	}
}
