package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ZenNotes/tui/internal/config"
)

// doctorWithTools runs `zn doctor --json` with config.toml holding terminal
// under [terminal] (none when empty) and PATH holding only stubs of tools.
func doctorWithTools(t *testing.T, terminal string, tools ...string) (string, []diagnosticCheck) {
	t.Helper()
	isolatedCLI(t)
	if terminal != "" {
		if err := os.MkdirAll(filepath.Dir(config.ConfigTomlPath()), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(config.ConfigTomlPath(), []byte("[terminal]\n"+terminal+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bin := t.TempDir()
	for _, tool := range tools {
		name := tool
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin)
	out := captureOutput(t, func() { _ = Main([]string{"doctor", "--json"}) })
	var report struct {
		Checks []diagnosticCheck `json:"checks"`
	}
	if err := json.Unmarshal([]byte(strings.SplitN(out, "\n{\"error\"", 2)[0]), &report); err != nil {
		t.Fatalf("doctor json: %v %s", err, out)
	}
	return out, report.Checks
}

func latexToolsCheck(t *testing.T, checks []diagnosticCheck, out string) diagnosticCheck {
	t.Helper()
	for _, c := range checks {
		if c.Name == "latex tools" {
			return c
		}
	}
	t.Fatalf("no \"latex tools\" check under math = \"latex\":\n%s", out)
	return diagnosticCheck{}
}

// AC75: under latex, a missing pdftex is a failing check that names it.
func TestDoctorLatexReportsMissingPdftex(t *testing.T) {
	out, checks := doctorWithTools(t, `math = "latex"`, "pdftocairo")
	c := latexToolsCheck(t, checks, out)
	if c.OK || !strings.Contains(c.Message, "pdftex") {
		t.Fatalf("AC75: latex tools check %+v", c)
	}
	if strings.Contains(c.Message, "pdftocairo") {
		t.Fatalf("AC75: pdftocairo is on PATH but reported missing: %+v", c)
	}
}

// AC76: under latex, a missing pdftocairo is a failing check that names it.
func TestDoctorLatexReportsMissingPdftocairo(t *testing.T) {
	out, checks := doctorWithTools(t, `math = "latex"`, "pdftex")
	c := latexToolsCheck(t, checks, out)
	if c.OK || !strings.Contains(c.Message, "pdftocairo") {
		t.Fatalf("AC76: latex tools check %+v", c)
	}
	if strings.Contains(strings.ReplaceAll(c.Message, "pdftocairo", ""), "pdftex") {
		t.Fatalf("AC76: pdftex is on PATH but reported missing: %+v", c)
	}
}

// AC77: under typst, or with no math key, doctor never mentions the TeX tools.
func TestDoctorTypstNeverMentionsTexTools(t *testing.T) {
	for _, terminal := range []string{`math = "typst"`, ``} {
		out, _ := doctorWithTools(t, terminal)
		for _, tool := range []string{"pdftex", "pdftocairo"} {
			if strings.Contains(out, tool) {
				t.Errorf("AC77 %q: doctor output mentions %s:\n%s", terminal, tool, out)
			}
		}
	}
}
