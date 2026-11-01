package compare

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/arcatva/groundwork/internal/judge"
	"github.com/arcatva/groundwork/internal/paths"
	"github.com/arcatva/groundwork/internal/text"
)

// Grades is how many times each PR and run is graded; a point counts as raised when most say so.
const Grades = 3

// Points groups the human review into distinct points, each with the full text of its threads, and
// caches them beside the PR.
func Points(pr PullRequest, work string) ([]Point, error) {
	cached := filepath.Join(pr.Folder, "points.json")
	var pts []Point
	if err := readJSON(cached, &pts); err == nil {
		return pts, nil
	}
	human, err := pr.Human()
	if err != nil {
		return nil, err
	}
	byID := map[string]Comment{}
	var ids []string
	for _, c := range human {
		byID[c.ID] = c
		ids = append(ids, c.ID)
	}
	sort.Strings(ids)
	var reply struct {
		Points []struct {
			ID      string   `json:"id"`
			Point   string   `json:"point"`
			Threads []string `json:"threads"`
		} `json:"points"`
		Trivial []string `json:"trivial"`
	}
	check := func() error {
		var named []string
		for _, p := range reply.Points {
			named = append(named, p.Threads...)
		}
		named = append(named, reply.Trivial...)
		sort.Strings(named)
		if !slices.Equal(named, ids) {
			return judge.Shape("each thread must be in exactly one point or in trivial")
		}
		return nil
	}
	comments, _ := json.MarshalIndent(human, "", " ")
	prompt := "Here is the review people gave a pull request: its thread-opening comments. Group them into " +
		"points: one point per distinct thing the reviewers ask to change (in behaviour, design, scope, " +
		"tests, naming or documentation content); comments that ask the same thing in different places " +
		"share a point. A comment that is only a typo, formatting or thanks is trivial. The repo is in " +
		"this directory; branch `pr` is the change, `base` is before it. Reply with only JSON: " +
		`{"points": [{"id": "P1", "point": "the ask in ten words", "threads": ["c1", "c3"]}], "trivial": ["c2"]}.` +
		"\n\n## Comments\n" + string(comments)
	if err := judge.AskJSON(prompt, &reply, check, judge.Options{Dir: work, Tools: "Read,Grep,Glob"}); err != nil {
		return nil, err
	}
	for _, p := range reply.Points {
		point := Point{ID: p.ID, Point: p.Point, Threads: p.Threads}
		for _, id := range p.Threads {
			point.Comments = append(point.Comments, byID[id])
		}
		pts = append(pts, point)
	}
	if pts == nil {
		pts = []Point{}
	}
	return pts, writeJSON(cached, pts)
}

// Anonymized is a report with every suite's name replaced, cut to what the grader is shown.
func Anonymized(report string) string {
	for _, s := range All {
		report = regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(s.Name)+`\b`).ReplaceAllString(report, "the skill")
	}
	return text.Head(report, 20000)
}

// Quoted says whether the quote's first words (four at least) appear, in order, in the report the
// grader saw. Matching words, not characters, survives the table pipes and arrows a report is written in.
func Quoted(quote string, report []string) bool {
	q := text.Tokens(quote)
	if len(q) > 12 {
		q = q[:12]
	}
	if len(q) < 4 {
		return false
	}
	for i := 0; i+len(q) <= len(report); i++ {
		if slices.Equal(report[i:i+len(q)], q) {
			return true
		}
	}
	return false
}

// InputsHash hashes what a grade depends on: every report and the points.
func InputsHash(records map[string]ReviewRecord, pts []Point) string {
	reports := map[string]string{}
	for name, r := range records {
		reports[name] = r.Report
	}
	b, _ := json.Marshal(struct {
		Reports map[string]string `json:"reports"`
		Points  []Point           `json:"points"`
	}{reports, pts})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:12]
}

func shuffled(names []string, seed string) []string {
	h := sha256.Sum256([]byte(seed))
	r := rand.New(rand.NewPCG(binary.BigEndian.Uint64(h[:8]), binary.BigEndian.Uint64(h[8:16])))
	out := slices.Clone(names)
	r.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

type gradedReport struct {
	Matches []struct {
		Point  string `json:"point"`
		Raised bool   `json:"raised"`
		Quote  string `json:"quote"`
	} `json:"matches"`
	OtherFindings []OtherFinding `json:"other_findings"`
}

// GradePRRun grades every suite's report for one PR and run, Grades times, blind and in a fresh
// random order each time, and keeps per point what most of the calls said.
func GradePRRun(pr PullRequest, runK string, records map[string]ReviewRecord, pts []Point, work string) (GradeFile, error) {
	meta, err := pr.Meta()
	if err != nil {
		return GradeFile{}, err
	}
	var names []string
	for name := range records {
		names = append(names, name)
	}
	sort.Strings(names)
	seen := map[string]string{}
	words := map[string][]string{}
	votes := map[string]map[string][]bool{}
	others := map[string][][]OtherFinding{}
	for _, name := range names {
		seen[name] = Anonymized(records[name].Report)
		words[name] = text.Tokens(seen[name])
		votes[name] = map[string][]bool{}
		for _, p := range pts {
			votes[name][p.ID] = []bool{}
		}
	}
	var ids []string
	for _, p := range pts {
		ids = append(ids, p.ID)
	}
	sort.Strings(ids)
	points, _ := json.MarshalIndent(pts, "", " ")
	for g := 0; g < Grades; g++ {
		order := shuffled(names, fmt.Sprintf("%s-%s-%d", pr.Number(), runK, g))
		letters := map[string]string{}
		var reports strings.Builder
		for i, name := range order {
			letter := string(rune('A' + i))
			letters[letter] = name
			fmt.Fprintf(&reports, "\n\n## Report %s\n%s", letter, seen[name])
		}
		reply := map[string]gradedReport{}
		check := func() error {
			for letter := range letters {
				got, ok := reply[letter]
				if !ok {
					return judge.Shape("no grade for report %s", letter)
				}
				var named []string
				for _, m := range got.Matches {
					named = append(named, m.Point)
				}
				sort.Strings(named)
				if !slices.Equal(named, ids) {
					return judge.Shape("report %s: not one verdict per point", letter)
				}
				for _, f := range got.OtherFindings {
					if f.Valid != "valid" && f.Valid != "invalid" && f.Valid != "unsure" {
						return judge.Shape("report %s: validity %q", letter, f.Valid)
					}
				}
			}
			return nil
		}
		prompt := "You grade several code reviews of one pull request against the review people gave it. The " +
			"repo is in this directory: branch `pr` is the change, `base` is before it; read the code to " +
			"judge. Each human point comes with the reviewers' own comments, their file and line. For " +
			"each report and each human point, say whether the report raised the same point (the same " +
			"problem at the same place, in any words) and quote one passage of the report that raises it, " +
			"copied exactly, with no ellipsis. Then list each finding in the report that matches no human " +
			"point, and judge it: valid (a problem this pull request causes or must fix), invalid (not a " +
			"problem, or not this pull request's) or unsure. The reports are lettered in a random order. " +
			"Reply with only JSON keyed by letter: " +
			`{"A": {"matches": [{"point": "P1", "raised": true, "quote": "..."}], ` +
			`"other_findings": [{"finding": "ten words", "valid": "valid", "why": "..."}]}}.` + "\n\n" +
			"## The pull request\n" + meta.Title + "\n\n" + meta.Body + "\n\n" +
			"## Human points\n" + string(points) + reports.String()
		if err := judge.AskJSON(prompt, &reply, check, judge.Options{Dir: work, Tools: "Read,Grep,Glob"}); err != nil {
			return GradeFile{}, err
		}
		for letter, name := range letters {
			for _, m := range reply[letter].Matches {
				// A "raised" counts only with a quote that is in the report the grader was shown.
				votes[name][m.Point] = append(votes[name][m.Point], m.Raised && Quoted(m.Quote, words[name]))
			}
			others[name] = append(others[name], reply[letter].OtherFindings)
		}
	}
	file := GradeFile{Inputs: InputsHash(records, pts), Suites: map[string]SuiteGrade{}}
	for _, name := range names {
		raised := map[string]bool{}
		for id, v := range votes[name] {
			yes := 0
			for _, b := range v {
				if b {
					yes++
				}
			}
			raised[id] = yes*2 > len(v)
		}
		file.Suites[name] = SuiteGrade{Raised: raised, Votes: votes[name], OtherFindings: others[name]}
	}
	return file, nil
}

// GradeAll grades each PR and run whose every suite finished clean, and regrades one whose inputs
// changed. A PR and run missing a clean review of any suite is not graded, since the suites are
// compared side by side.
func GradeAll(layout paths.Layout, results string) error {
	files, err := filepath.Glob(filepath.Join(results, "*-r*.json"))
	if err != nil {
		return err
	}
	type key struct{ pr, run string }
	groups := map[key]map[string]ReviewRecord{}
	for _, path := range files {
		if strings.HasSuffix(path, ".grade.json") {
			continue
		}
		var rec ReviewRecord
		if err := readJSON(path, &rec); err != nil {
			return err
		}
		if !rec.Clean() {
			continue
		}
		stem := strings.TrimSuffix(filepath.Base(path), ".json")
		k := key{rec.PR, stem[strings.LastIndex(stem, "-r")+2:]}
		if groups[k] == nil {
			groups[k] = map[string]ReviewRecord{}
		}
		groups[k][rec.Skill] = rec
	}
	var suites Suites
	if err := readJSON(filepath.Join(results, "suites.json"), &suites); err != nil {
		return err
	}
	keys := make([]key, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].pr+"-"+keys[i].run < keys[j].pr+"-"+keys[j].run })
	for _, k := range keys {
		records := groups[k]
		var missing []string
		for _, name := range suites.Names {
			if _, ok := records[name]; !ok {
				missing = append(missing, name)
			}
		}
		if len(missing) > 0 {
			fmt.Printf("INCOMPLETE #%s run %s: no clean run of %s; not graded\n", k.pr, k.run, strings.Join(missing, ", "))
			continue
		}
		if err := gradeGroup(layout, results, k.pr, k.run, records); err != nil {
			if !errors.Is(err, judge.ErrShape) {
				return err
			}
			fmt.Printf("UNGRADED #%s run %s: %v\n", k.pr, k.run, err)
		}
	}
	return nil
}

func gradeGroup(layout paths.Layout, results, number, runK string, records map[string]ReviewRecord) error {
	out := filepath.Join(results, fmt.Sprintf("%s-r%s.grade.json", number, runK))
	pr := NamedPR(layout, number)
	tmp, err := os.MkdirTemp("", "grade-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	work := filepath.Join(tmp, "repo")
	if err := pr.Workspace(work); err != nil {
		return err
	}
	pts, err := Points(pr, work)
	if err != nil {
		return err
	}
	var existing GradeFile
	if readJSON(out, &existing) == nil && existing.Inputs == InputsHash(records, pts) {
		return nil
	}
	file, err := GradePRRun(pr, runK, records, pts, work)
	if err != nil {
		return err
	}
	fmt.Printf("graded #%s run %s: %d reports, %d points\n", number, runK, len(records), len(pts))
	return writeJSON(out, file)
}
