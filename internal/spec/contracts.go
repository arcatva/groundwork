package spec

import (
	"fmt"
	"strings"
)

// Finding is one problem a reviewer reports.
type Finding struct {
	Where   string `json:"where" check:"required" desc:"file:line; for something missing, the file where it belongs; for a claim, the PR description"`
	Problem string `json:"problem" check:"required" desc:"what is wrong"`
	Fix     string `json:"fix" check:"required" desc:"the most direct fix first"`
	RestsOn string `json:"rests_on" check:"required" desc:"its evidence, or a source and its rank"`
}

// Option is one way to settle a question.
type Option struct {
	Letter  string `json:"letter" check:"required" desc:"a, b, c..."`
	Option  string `json:"option" check:"required"`
	Example string `json:"example" check:"required" desc:"what happens, on a concrete value"`
}

// Question is a product choice the code cannot settle, put to whoever decides.
type Question struct {
	Number    int      `json:"number" check:"required"`
	Decision  string   `json:"decision" check:"required" desc:"the choice to make"`
	Now       string   `json:"now" check:"required" desc:"what the code does now, with path:line"`
	Sources   string   `json:"sources" check:"required" desc:"what the ticket, design or a decision says, quoted and linked, or that nothing was found"`
	Options   []Option `json:"options" check:"min=2"`
	Recommend string   `json:"recommend" check:"required" desc:"the letter you recommend, even with no source"`
	Why       string   `json:"why" check:"required"`
}

// Check holds the recommendation to one of the options.
func (q Question) Check() []error {
	for _, o := range q.Options {
		if o.Letter == q.Recommend {
			return nil
		}
	}
	return []error{fmt.Errorf("recommends %q, which is not one of its options", q.Recommend)}
}

// Name is one entry of the Design reviewer's name inventory.
type Name struct {
	Name      string `json:"name" check:"required"`
	Holds     string `json:"holds" check:"required" desc:"what it holds or does"`
	Elsewhere string `json:"elsewhere" check:"required" desc:"how the repo names the same role elsewhere, with a git grep count"`
	Clash     string `json:"clash" desc:"what the word already means elsewhere, if anything"`
}

// ReviewerReport is what one reviewer hands back: its findings, its questions, and for Design, the
// names the diff adds.
type ReviewerReport struct {
	Reviewer      string     `json:"reviewer" check:"required,enum=Claims|Behaviour|Design"`
	Findings      []Finding  `json:"findings"`
	Questions     []Question `json:"questions"`
	NameInventory []Name     `json:"name_inventory" desc:"Design only: every name the diff adds"`
}

// Check requires Design's name inventory, and only Design's.
func (r ReviewerReport) Check() []error {
	switch {
	case r.Reviewer == "Design" && len(r.NameInventory) == 0:
		return []error{fmt.Errorf("the Design report has no name inventory")}
	case r.Reviewer != "Design" && len(r.NameInventory) > 0:
		return []error{fmt.Errorf("only the Design report holds a name inventory")}
	}
	return nil
}

// Answer is something the user said, quoted.
type Answer struct {
	To     string `json:"to" check:"required" desc:"the question or proposal it answers"`
	Quoted string `json:"quoted" check:"required" desc:"the user's own words"`
}

// Skip names a step the flow went past, and the user's word for it.
type Skip struct {
	Step    string `json:"step" check:"required"`
	Because string `json:"because" check:"required" desc:"the user's words, quoted"`
}

// LogEntry is one line of the run log, `.groundwork/log.jsonl`.
type LogEntry struct {
	Time        string   `json:"time" check:"required" desc:"RFC 3339"`
	Branch      string   `json:"branch" check:"required"`
	Commit      string   `json:"commit" check:"required"`
	Version     string   `json:"groundwork_version" check:"required" desc:"the number in SKILL.md's heading"`
	Step        string   `json:"step" check:"required,enum=Frame|Plan|Build|Review|Ship|Learn"`
	Status      string   `json:"status" check:"required,enum=done|needs you|blocked"`
	Report      string   `json:"report" check:"required" desc:"the report the user saw"`
	Gate        string   `json:"gate" desc:"the gate's output"`
	Answers     []Answer `json:"answers"`
	Skipped     []Skip   `json:"skipped" desc:"steps the flow went past, each with the user's word for it"`
	Corrections []string `json:"corrections" desc:"a step skipped, a gate passed without its output, a contract broken"`
}

// Check holds every skipped step to a name in the flow.
func (e LogEntry) Check() []error {
	var errs []error
	for _, s := range e.Skipped {
		if !contains(StepNames, s.Step) {
			errs = append(errs, fmt.Errorf("skips %q, which is no step (%s)", s.Step, strings.Join(StepNames, ", ")))
		}
	}
	return errs
}

// StepNames are the flow's steps, in order.
var StepNames = []string{"Frame", "Plan", "Build", "Review", "Ship", "Learn"}
