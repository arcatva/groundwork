// Package compare compares review skills on public pull requests, replayed at the commit the first
// human review saw, and grades every report blind against the points the human reviewers made.
package compare

import (
	"encoding/json"
	"os"
)

// Meta is a prepared PR's description and the two commits it is replayed between.
type Meta struct {
	Repo        string `json:"repo"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	Base        string `json:"base"`
	Head        string `json:"head"`
	FirstReview string `json:"first_review"`
	BaseMessage string `json:"base_message"`
	HeadMessage string `json:"head_message"`
}

// Comment is one thread-opening human review comment on the reviewed head, or a review's body.
type Comment struct {
	ID   string  `json:"id"`
	Path *string `json:"path"`
	Line *int    `json:"line"`
	Body string  `json:"body"`
}

// Point is one distinct ask of the human review, with the comments that make it.
type Point struct {
	ID       string    `json:"id"`
	Point    string    `json:"point"`
	Threads  []string  `json:"threads"`
	Comments []Comment `json:"comments"`
}

// ReviewRecord is one suite's review of one PR: how it ended, what it cost, whether it read the human
// review or tripped over another run's files, and its report, which is what the user would see.
type ReviewRecord struct {
	Skill            string         `json:"skill"`
	PR               string         `json:"pr"`
	Init             map[string]any `json:"init"`
	Minutes          float64        `json:"minutes"`
	CostUSD          *float64       `json:"cost_usd"`
	Finished         bool           `json:"finished"`
	Reason           *string        `json:"reason"`
	Contaminated     bool           `json:"contaminated"`
	ReachedForHidden bool           `json:"reached_for_hidden"`
	Collided         bool           `json:"collided"`
	LeftInSharedTmp  []string       `json:"left_in_shared_tmp"`
	Report           string         `json:"report"`
	Trace            string         `json:"trace"`
}

// Clean says whether a review counts: it finished, and neither read the human review nor tripped over
// another run's files.
func (r ReviewRecord) Clean() bool {
	return r.Finished && !r.Contaminated && !r.Collided
}

// Why says why a review does not count.
func (r ReviewRecord) Why() string {
	switch {
	case r.Contaminated:
		return "contaminated"
	case r.Collided:
		return "collided"
	case r.Reason != nil:
		return *r.Reason
	}
	return "did not finish"
}

// Origin says where a suite's text came from.
type Origin struct {
	Source      string `json:"source"`
	Commit      string `json:"commit"`
	SkillSHA256 string `json:"skill_sha256"`
}

// Suites is what a comparison run recorded before its first review: the suites and their origins.
type Suites struct {
	Names   []string          `json:"names"`
	Origins map[string]Origin `json:"origins"`
	Budget  float64           `json:"budget"`
	Runs    int               `json:"runs"`
}

// OtherFinding is a finding that matches no human point, with the grader's judgement of it.
type OtherFinding struct {
	Finding string `json:"finding"`
	Valid   string `json:"valid"` // valid, invalid or unsure
	Why     string `json:"why"`
}

// SuiteGrade is one suite's grade on one PR and run: per point, whether most grader calls said it was
// raised, every call's vote, and every call's list of other findings.
type SuiteGrade struct {
	Raised        map[string]bool   `json:"raised"`
	Votes         map[string][]bool `json:"votes"`
	OtherFindings [][]OtherFinding  `json:"other_findings"`
}

// GradeFile is the grade of every suite on one PR and run. Inputs hashes what was graded, so a grade
// is redone when its inputs change.
type GradeFile struct {
	Inputs string                `json:"inputs"`
	Suites map[string]SuiteGrade `json:"suites"`
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
