// Package sandbox is the sandbox every run gets, and the temp folder runs share.
//
// Each run is headless Claude Code with its shell in Claude Code's sandbox: no network, writes only in
// its work tree and the sandbox's temp folder, and anything that would need a prompt denied, since
// nobody is there to answer one. The sandbox gives every run the same temp folder ($TMPDIR is always
// /tmp/claude-<uid>, whatever the run's environment or settings say), so runs that must not see each
// other go one at a time: what was in that folder when a run started is unreadable to it, and what it
// leaves there is moved out afterwards.
package sandbox

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/arcatva/groundwork/internal/fsx"
)

// Tools are the tools a run may use.
var Tools = []string{"Bash", "Read", "Edit", "Write", "Grep", "Glob", "Agent", "Skill"}

// claudeOwn prefixes Claude Code's own entries in the shared temp folder (session folders, caches),
// which runs leave alone.
var claudeOwn = []string{"-", "cache", "bash-edit-diff"}

// Settings is the sandbox settings, with hidden paths unreadable to the shell and to the Read tool.
func Settings(hidden []string) map[string]any {
	sandbox := map[string]any{
		"enabled":                  true,
		"autoAllowBashIfSandboxed": true,
		"allowUnsandboxedCommands": false,
		"failIfUnavailable":        true,
		"network":                  map[string]any{"allowedDomains": []string{}},
	}
	s := map[string]any{
		"sandbox": sandbox,
		// Hooks run outside the sandbox; a run loads none it can be told not to.
		"disableAllHooks": true,
	}
	paths := unique(hidden)
	if len(paths) > 0 {
		sandbox["filesystem"] = map[string]any{"denyRead": paths}
		deny := make([]string, len(paths))
		for i, p := range paths {
			deny[i] = "Read(/" + p + "/**)"
		}
		s["permissions"] = map[string]any{"deny": deny}
	}
	return s
}

var collision = regexp.MustCompile(`/claude-\d+/\S*?['"]? already exists`)

// Collided says whether a run tripped over something already in the shared temp folder.
func Collided(output string) bool {
	return collision.MatchString(output)
}

// SharedTemp is the temp folder the sandbox gives every run.
type SharedTemp struct {
	Root    string
	Archive string
}

// NewSharedTemp is the current user's sandbox temp folder, archiving leftovers into archive.
func NewSharedTemp(archive string) SharedTemp {
	return SharedTemp{Root: filepath.Join("/tmp", fmt.Sprintf("claude-%d", os.Getuid())), Archive: archive}
}

// Entries is the names in the folder now.
func (s SharedTemp) Entries() map[string]bool {
	names := map[string]bool{}
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		return names
	}
	for _, e := range entries {
		names[e.Name()] = true
	}
	return names
}

// Stale is the paths of what was there before a run, under both spellings of the folder.
func (s SharedTemp) Stale(existing map[string]bool) []string {
	roots := []string{s.Root}
	if real, err := filepath.EvalSymlinks(s.Root); err == nil && real != s.Root {
		roots = append(roots, real)
	}
	var paths []string
	for name := range existing {
		for _, root := range roots {
			paths = append(paths, filepath.Join(root, name))
		}
	}
	sort.Strings(paths)
	return paths
}

// Tidy moves what a run left in the folder to the archive, so the next run cannot trip over it or read
// it, and returns what it moved. Moving, not deleting: the archive is the record of what each run wrote.
func (s SharedTemp) Tidy(created map[string]bool, runID string) ([]string, error) {
	var names []string
	for name := range created {
		names = append(names, name)
	}
	sort.Strings(names)
	var moved []string
	for _, name := range names {
		if ownedByClaude(name) {
			continue
		}
		src := filepath.Join(s.Root, name)
		if _, err := os.Lstat(src); err != nil {
			continue
		}
		dest := filepath.Join(s.Archive, runID)
		if err := os.MkdirAll(dest, 0o755); err != nil {
			return moved, err
		}
		if err := fsx.Move(src, filepath.Join(dest, name)); err != nil {
			return moved, err
		}
		moved = append(moved, name)
	}
	return moved, nil
}

// Diff is the names in now that were not in before.
func Diff(now, before map[string]bool) map[string]bool {
	out := map[string]bool{}
	for name := range now {
		if !before[name] {
			out[name] = true
		}
	}
	return out
}

func ownedByClaude(name string) bool {
	for _, prefix := range claudeOwn {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func unique(xs []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Strings(out)
	return out
}
