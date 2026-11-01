package lint

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcatva/groundwork/internal/fsx"
	"github.com/arcatva/groundwork/internal/paths"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// TestRepo is the gate CI runs: the repo as it is breaks none of its own rules.
func TestRepo(t *testing.T) {
	for _, p := range Run(paths.At(repoRoot(t))) {
		t.Error(p)
	}
}

// copyRepo copies what the checks read into a fresh git repo, so a test can break it.
func copyRepo(t *testing.T) paths.Layout {
	t.Helper()
	src, dst := repoRoot(t), t.TempDir()
	for _, part := range []string{"skills", "evals/scenarios", ".claude-plugin", ".codex-plugin", ".cursor-plugin", ".kimi-plugin", ".devin-plugin", ".grok-plugin", ".agents", "README.md"} {
		from := filepath.Join(src, part)
		if _, err := os.Stat(from); err != nil {
			continue
		}
		if err := fsx.CopyTree(from, filepath.Join(dst, part), nil); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{{"init", "-q"}, {"add", "-A"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dst
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	return paths.At(dst)
}

func edit(t *testing.T, path string, change func(string) string) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(change(string(b))), 0o644); err != nil {
		t.Fatal(err)
	}
}

func appendTo(line string) func(string) string {
	return func(s string) string { return s + "\n" + line + "\n" }
}

func TestPrivateWordsMatchWholeWords(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".neutral-words"), []byte("acme\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	re, err := privateWords(root)
	if err != nil || re == nil {
		t.Fatal(err)
	}
	if !re.MatchString("Built at Acme.") {
		t.Error("the word itself was missed")
	}
	if re.MatchString("TestAcmeishSplit") {
		t.Error("a word that only contains it was flagged")
	}
}

// TestEachCheckCatchesItsBreak breaks each rule once, in a copy of the repo, and expects the check
// that owns the rule to say so: a check that passes on a broken repo proves nothing.
func TestEachCheckCatchesItsBreak(t *testing.T) {
	cases := []struct {
		name, check string
		breakIt     func(t *testing.T, l paths.Layout)
	}{
		{"vendor named in the skill", "neutral", func(t *testing.T, l paths.Layout) {
			edit(t, filepath.Join(l.Skill, "flow", "build.md"), appendTo("Check the Figma frame."))
		}},
		{"agent named outside a file name", "neutral", func(t *testing.T, l paths.Layout) {
			edit(t, filepath.Join(l.Skill, "flow", "build.md"), appendTo("Ask Claude to run it."))
		}},
		{"a pull request called an MR", "neutral", func(t *testing.T, l paths.Layout) {
			edit(t, filepath.Join(l.Skill, "flow", "ship.md"), appendTo("Open the MR."))
		}},
		{"a private word in a tracked file", "neutral", func(t *testing.T, l paths.Layout) {
			if err := os.WriteFile(filepath.Join(l.Root, ".neutral-words"), []byte("acme-internal\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			edit(t, filepath.Join(l.Root, "README.md"), appendTo("Built at acme-internal."))
		}},
		{"a normative keyword in the flow", "structure", func(t *testing.T, l paths.Layout) {
			edit(t, filepath.Join(l.Skill, "flow", "build.md"), appendTo("You MUST do this."))
		}},
		{"a pointer to a missing rule file", "structure", func(t *testing.T, l paths.Layout) {
			edit(t, filepath.Join(l.Skill, "flow", "plan.md"), appendTo("See `standard/functional/nowhere.md`."))
		}},
		{"a rule file with no reviewer", "structure", func(t *testing.T, l paths.Layout) {
			if err := os.WriteFile(filepath.Join(l.Skill, "standard", "functional", "orphan.md"), []byte("# Orphan\n"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"a section quoted that does not exist", "lean", func(t *testing.T, l paths.Layout) {
			edit(t, filepath.Join(l.Skill, "flow", "build.md"), appendTo(`As "Facts" in `+"`SKILL.md`"+` says.`))
		}},
		{"the word budget exceeded", "lean", func(t *testing.T, l paths.Layout) {
			edit(t, filepath.Join(l.Skill, "flow", "build.md"), appendTo(strings.Repeat("word ", WordBudget)))
		}},
		{"a generated table edited by hand", "generated", func(t *testing.T, l paths.Layout) {
			edit(t, filepath.Join(l.Skill, "SKILL.md"), func(s string) string {
				return strings.Replace(s, "| a call or a signature |", "| a call |", 1)
			})
		}},
		{"a manifest's version out of step", "manifests", func(t *testing.T, l paths.Layout) {
			edit(t, filepath.Join(l.Root, ".codex-plugin", "plugin.json"), func(s string) string {
				return strings.Replace(s, `"version": "`, `"version": "9`, 1)
			})
		}},
		{"a scenario breaking its contract", "scenarios", func(t *testing.T, l paths.Layout) {
			dir := filepath.Join(l.Scenarios, "broken")
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, "scenario.json"), []byte(`{"prompt": "x", "rubric": []}`), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			layout := copyRepo(t)
			if before := Run(layout); len(before) > 0 {
				t.Fatalf("the copy is broken before the change: %v", before)
			}
			c.breakIt(t, layout)
			if c.check == "neutral" {
				// A new file is tracked only once added.
				cmd := exec.Command("git", "add", "-A")
				cmd.Dir = layout.Root
				_ = cmd.Run()
			}
			for _, p := range Run(layout) {
				if p.Check == c.check {
					return
				}
			}
			t.Errorf("check %q did not catch: %s", c.check, c.name)
		})
	}
}
