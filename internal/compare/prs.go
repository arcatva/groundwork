package compare

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/arcatva/groundwork/internal/gitx"
	"github.com/arcatva/groundwork/internal/paths"
)

// PullRequest is a prepared PR: meta.json, human.json (the thread-opening review comments on the
// reviewed head), points.json once graded, and both trees as tarballs.
type PullRequest struct {
	Folder string
	layout paths.Layout
}

// NamedPR is the prepared PR with this number.
func NamedPR(layout paths.Layout, number string) PullRequest {
	return PullRequest{Folder: filepath.Join(layout.PRs, number), layout: layout}
}

// Number is the PR's number.
func (p PullRequest) Number() string { return filepath.Base(p.Folder) }

// Meta reads the PR's description and commits.
func (p PullRequest) Meta() (Meta, error) {
	var m Meta
	return m, readJSON(filepath.Join(p.Folder, "meta.json"), &m)
}

// Human reads the human review comments on the reviewed head.
func (p PullRequest) Human() ([]Comment, error) {
	var c []Comment
	return c, readJSON(filepath.Join(p.Folder, "human.json"), &c)
}

// Task is the same request every reviewer gets.
func (p PullRequest) Task() (string, error) {
	m, err := p.Meta()
	if err != nil {
		return "", err
	}
	return "Review the pull request on branch `pr` against branch `base` in this repo. " +
		"A Python with the repo's test dependencies is at .venv/bin/python. " +
		"Report your findings to me. Do not commit or change the branches.\n\n" +
		"Its title: " + m.Title + "\n\nIts description:\n" + m.Body, nil
}

// Workspace builds the repo at base and at the reviewed head as two branches with their real commit
// messages and no other history, so the commits that answered the review cannot leak in.
func (p PullRequest) Workspace(dest string) error {
	m, _ := p.Meta()
	baseMessage, headMessage := orDefault(m.BaseMessage, "base"), orDefault(m.HeadMessage, "pr")
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if err := Unpack(filepath.Join(p.Folder, "base.tgz"), dest); err != nil {
		return err
	}
	for _, step := range [][]string{{"init", "-q", "-b", "base"}, {"add", "-A"}, {"commit", "-qm", baseMessage}, {"checkout", "-qb", "pr"}} {
		if _, err := gitx.Git(dest, step...); err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Name() != ".git" {
			if err := os.RemoveAll(filepath.Join(dest, e.Name())); err != nil {
				return err
			}
		}
	}
	if err := Unpack(filepath.Join(p.Folder, "head.tgz"), dest); err != nil {
		return err
	}
	for _, step := range [][]string{{"add", "-A"}, {"commit", "-qm", headMessage}} {
		if _, err := gitx.Git(dest, step...); err != nil {
			return err
		}
	}
	if err := os.Symlink(p.layout.Venv, filepath.Join(dest, ".venv")); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dest, ".git", "info"), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dest, ".git", "info", "exclude"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(".venv\n")
	return err
}

func orDefault(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}

// Unpack extracts a GitHub tarball into dest without its top folder. Like a careful `tar`: no
// absolute paths, nothing outside dest, no device files. A repo's own agent settings (.claude/,
// .mcp.json) are dropped, since they could add hooks or MCP servers to the run; its instruction files
// (AGENTS.md, CLAUDE.md) stay, as they are the repo's written rules.
func Unpack(tarball, dest string) error {
	f, err := os.Open(tarball)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}
		rel := h.Name
		if i := strings.Index(rel, "/"); i >= 0 {
			rel = rel[i+1:]
		} else {
			rel = ""
		}
		if rel == "" || h.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		if filepath.IsAbs(rel) || strings.Contains("/"+rel+"/", "/../") {
			return fmt.Errorf("unsafe path in %s: %q", tarball, h.Name)
		}
		if agentConfig(rel) || (h.Typeflag == tar.TypeSymlink && agentConfig(h.Linkname)) {
			continue
		}
		target := filepath.Join(dest, rel)
		// Never write through a symlink an earlier entry made: a chain of in-tree links can still
		// point out of the tree.
		if throughLink(dest, rel) {
			return fmt.Errorf("unsafe path in %s: %q passes through a symlink", tarball, h.Name)
		}
		switch h.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fs.FileMode(h.Mode)&0o755|0o600)
			if err != nil {
				return err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				return err
			}
			if err := out.Close(); err != nil {
				return err
			}
		case tar.TypeSymlink:
			// A link with an absolute target or any ".." step is left out: where it really points depends on
			// links met on the way, which the text alone cannot tell.
			if filepath.IsAbs(h.Linkname) || slices.Contains(strings.Split(filepath.ToSlash(h.Linkname), "/"), "..") {
				continue
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Symlink(h.Linkname, target); err != nil {
				return err
			}
		}
	}
	return nil
}

// agentConfig says whether a path holds a repo's agent settings or git's own folder, whose hooks would
// run outside the sandbox. Case is ignored, since a case-insensitive file system ignores it too.
func agentConfig(rel string) bool {
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		for _, name := range []string{".claude", ".mcp.json", ".git"} {
			if strings.EqualFold(part, name) {
				return true
			}
		}
	}
	return false
}

// throughLink says whether any folder above rel inside dest is a symlink.
func throughLink(dest, rel string) bool {
	parts := strings.Split(filepath.ToSlash(filepath.Dir(rel)), "/")
	at := dest
	for _, part := range parts {
		if part == "." || part == "" {
			continue
		}
		at = filepath.Join(at, part)
		if info, err := os.Lstat(at); err == nil && info.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

const bodyEdits = `query($owner: String!, $name: String!, $n: Int!) {
  repository(owner: $owner, name: $name) { pullRequest(number: $n) {
    userContentEdits(first: 50) { nodes { editedAt diff } } } } }`

// BodyAt is the description as it stood at `when`: the last edit made by then, or the current text.
func BodyAt(repo string, n int, current, when string) (string, error) {
	owner, name, _ := strings.Cut(repo, "/")
	var reply struct {
		Data struct {
			Repository struct {
				PullRequest struct {
					UserContentEdits struct {
						Nodes []struct {
							EditedAt string  `json:"editedAt"`
							Diff     *string `json:"diff"`
						} `json:"nodes"`
					} `json:"userContentEdits"`
				} `json:"pullRequest"`
			} `json:"repository"`
		} `json:"data"`
	}
	vars := map[string]string{"owner": owner, "name": name, "n": strconv.Itoa(n)}
	if err := gitx.GraphQL(bodyEdits, vars, &reply); err != nil {
		return "", err
	}
	best, bestAt := current, ""
	for _, e := range reply.Data.Repository.PullRequest.UserContentEdits.Nodes {
		// RFC 3339 timestamps in one zone compare as strings.
		if e.Diff != nil && e.EditedAt <= when && e.EditedAt > bestAt {
			best, bestAt = *e.Diff, e.EditedAt
		}
	}
	return best, nil
}

type ghUser struct {
	Login string `json:"login"`
	Type  string `json:"type"`
}

type ghComment struct {
	User             ghUser `json:"user"`
	Path             string `json:"path"`
	OriginalLine     *int   `json:"original_line"`
	OriginalCommitID string `json:"original_commit_id"`
	CreatedAt        string `json:"created_at"`
	InReplyToID      *int64 `json:"in_reply_to_id"`
	Body             string `json:"body"`
}

type ghReview struct {
	User     ghUser `json:"user"`
	CommitID string `json:"commit_id"`
	Body     string `json:"body"`
}

// Prepare fetches each PR: its description as at the first human review, the thread-opening review
// comments on the commit that review saw, and both trees. It also makes the Python environment the
// reviewers get, and clones the rival suites.
func Prepare(layout paths.Layout, repo string, numbers []int) error {
	for _, n := range numbers {
		if err := prepareOne(layout, repo, n); err != nil {
			return fmt.Errorf("#%d: %w", n, err)
		}
	}
	if _, err := os.Stat(layout.Venv); err != nil {
		if err := exec.Command("python3", "-m", "venv", layout.Venv).Run(); err != nil {
			return err
		}
		// Django's test runner needs these; other repos would list theirs here.
		if err := exec.Command(filepath.Join(layout.Venv, "bin", "pip"), "install", "-q", "asgiref", "sqlparse").Run(); err != nil {
			return err
		}
	}
	for _, s := range All {
		dest := filepath.Join(layout.Rivals, s.Name)
		if s.Repo == "" {
			continue
		}
		if _, err := os.Stat(dest); err == nil {
			continue
		}
		if err := exec.Command("gh", "repo", "clone", s.Repo, dest, "--", "--depth", "1", "-q").Run(); err != nil {
			return err
		}
	}
	return nil
}

func prepareOne(layout paths.Layout, repo string, n int) error {
	out := filepath.Join(layout.PRs, strconv.Itoa(n))
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	var pr struct {
		Title string  `json:"title"`
		Body  *string `json:"body"`
		User  ghUser  `json:"user"`
		Base  struct {
			Ref string `json:"ref"`
		} `json:"base"`
	}
	if err := gitx.GH(fmt.Sprintf("repos/%s/pulls/%d", repo, n), false, &pr); err != nil {
		return err
	}
	var comments []ghComment
	if err := gitx.GH(fmt.Sprintf("repos/%s/pulls/%d/comments?per_page=100", repo, n), true, &comments); err != nil {
		return err
	}
	// Thread-opening comments only: a reply ("Ditto", "done") is not a point of its own.
	var roots []ghComment
	for _, c := range comments {
		if c.User.Login != pr.User.Login && c.User.Type != "Bot" && c.InReplyToID == nil {
			roots = append(roots, c)
		}
	}
	if len(roots) == 0 {
		fmt.Printf("#%d: no human inline review, skipped\n", n)
		return nil
	}
	sort.SliceStable(roots, func(i, j int) bool { return roots[i].CreatedAt < roots[j].CreatedAt })
	reviewed, when := roots[0].OriginalCommitID, roots[0].CreatedAt
	var cmp struct {
		MergeBase struct {
			SHA string `json:"sha"`
		} `json:"merge_base_commit"`
	}
	if err := gitx.GH(fmt.Sprintf("repos/%s/compare/%s...%s", repo, pr.Base.Ref, reviewed), false, &cmp); err != nil {
		return err
	}
	var reviews []ghReview
	if err := gitx.GH(fmt.Sprintf("repos/%s/pulls/%d/reviews?per_page=100", repo, n), true, &reviews); err != nil {
		return err
	}
	var human []Comment
	i := 0
	for _, c := range roots {
		if c.OriginalCommitID == reviewed {
			i++
			path := c.Path
			human = append(human, Comment{ID: fmt.Sprintf("c%d", i), Path: &path, Line: c.OriginalLine, Body: c.Body})
		}
	}
	j := 0
	for _, r := range reviews {
		if r.User.Login != pr.User.Login && r.CommitID == reviewed && strings.TrimSpace(r.Body) != "" {
			j++
			human = append(human, Comment{ID: fmt.Sprintf("r%d", j), Body: r.Body})
		}
	}
	body := ""
	if pr.Body != nil {
		body = *pr.Body
	}
	body, err := BodyAt(repo, n, body, when)
	if err != nil {
		return err
	}
	message := func(sha string) (string, error) {
		var c struct {
			Commit struct {
				Message string `json:"message"`
			} `json:"commit"`
		}
		return c.Commit.Message, gitx.GH(fmt.Sprintf("repos/%s/commits/%s", repo, sha), false, &c)
	}
	baseMessage, err := message(cmp.MergeBase.SHA)
	if err != nil {
		return err
	}
	headMessage, err := message(reviewed)
	if err != nil {
		return err
	}
	meta := Meta{repo, n, pr.Title, body, cmp.MergeBase.SHA, reviewed, when, baseMessage, headMessage}
	if err := writeJSON(filepath.Join(out, "meta.json"), meta); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(out, "human.json"), human); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(out, "points.json"))
	for name, sha := range map[string]string{"base": cmp.MergeBase.SHA, "head": reviewed} {
		tarball := filepath.Join(out, name+".tgz")
		if _, err := os.Stat(tarball); err == nil {
			continue
		}
		if err := gitx.Download(fmt.Sprintf("repos/%s/tarball/%s", repo, sha), tarball); err != nil {
			return err
		}
	}
	fmt.Printf("#%d: %d human threads on %.10s, base %.10s\n", n, len(human), reviewed, cmp.MergeBase.SHA)
	return nil
}
