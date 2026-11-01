// Package gitx runs git and the GitHub CLI the way the harness needs them.
package gitx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// A fixed identity, and no background gc: a detached `git gc` still writing .git would break the
// cleanup of the temp folder the repo lives in.
var config = []string{
	"-c", "user.name=harness", "-c", "user.email=harness@example.com",
	"-c", "gc.auto=0", "-c", "maintenance.auto=false",
	// No hooks and no fsmonitor: a work tree built from an untrusted tarball must not run its code.
	"-c", "core.hooksPath=/dev/null", "-c", "core.fsmonitor=false",
}

// Git runs git in dir and returns its output.
func Git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", append(append([]string{}, config...), args...)...)
	cmd.Dir = dir
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(errOut.String()))
	}
	return out.String(), nil
}

// Head is the short commit at HEAD in dir, or "" when there is none.
func Head(dir string) string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	cmd.Dir = dir
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

var pageJoin = regexp.MustCompile(`\]\s*\[`)

// GH calls the GitHub API through `gh api` and decodes the reply into v. With pages, every page is
// fetched and their arrays joined.
func GH(path string, pages bool, v any) error {
	args := []string{"api"}
	if pages {
		args = append(args, "--paginate")
	}
	out, err := exec.Command("gh", append(args, path)...).Output()
	if err != nil {
		return fmt.Errorf("gh api %s: %w", path, err)
	}
	if pages {
		// --paginate concatenates the pages' JSON arrays as `][`; join them into one.
		out = pageJoin.ReplaceAll(out, []byte(","))
	}
	return json.Unmarshal(out, v)
}

// GraphQL runs a GitHub GraphQL query with string variables and decodes the reply into v.
func GraphQL(query string, vars map[string]string, v any) error {
	args := []string{"api", "graphql", "-f", "query=" + query}
	for name, value := range vars {
		args = append(args, "-F", name+"="+value)
	}
	out, err := exec.Command("gh", args...).Output()
	if err != nil {
		return fmt.Errorf("gh api graphql: %w", err)
	}
	return json.Unmarshal(out, v)
}

// Download saves what a GitHub API path returns (a tarball, say) to dest.
func Download(apiPath, dest string) error {
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	cmd := exec.Command("gh", "api", apiPath)
	cmd.Stdout = f
	return cmd.Run()
}
