package compare

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/arcatva/groundwork/internal/paths"
	"github.com/arcatva/groundwork/internal/text"
)

// Report renders the comparison as markdown: per suite, the share of human points raised over the PRs
// every suite was graded on (per PR first, then averaged), the run-to-run spread, other findings and
// cost per review; then each PR's points and who raised them; then every review left out, with why.
func Report(layout paths.Layout, results string) (string, error) {
	gradeFiles, err := filepath.Glob(filepath.Join(results, "*.grade.json"))
	if err != nil {
		return "", err
	}
	graded := map[string]map[string]map[string]SuiteGrade{} // pr -> run -> suite
	suiteSet := map[string]bool{}
	for _, path := range gradeFiles {
		number, run, _ := strings.Cut(strings.TrimSuffix(filepath.Base(path), ".grade.json"), "-r")
		var file GradeFile
		if err := readJSON(path, &file); err != nil {
			return "", err
		}
		if graded[number] == nil {
			graded[number] = map[string]map[string]SuiteGrade{}
		}
		graded[number][run] = file.Suites
		for name := range file.Suites {
			suiteSet[name] = true
		}
	}
	suites := sortedKeys(suiteSet)
	pts := map[string][]Point{}
	var noPoints, common []string
	for number := range graded {
		var p []Point
		if err := readJSON(filepath.Join(NamedPR(layout, number).Folder, "points.json"), &p); err != nil {
			return "", err
		}
		pts[number] = p
		if len(p) == 0 {
			noPoints = append(noPoints, number)
			continue
		}
		everySuite := true
		for _, s := range suites {
			found := false
			for _, g := range graded[number] {
				if _, ok := g[s]; ok {
					found = true
				}
			}
			everySuite = everySuite && found
		}
		if everySuite {
			common = append(common, number)
		}
	}
	sort.Strings(noPoints)
	sort.Strings(common)

	perPR := func(number, suite string) []float64 {
		var shares []float64
		for _, g := range graded[number] {
			if sg, ok := g[suite]; ok {
				raised := 0.0
				for _, v := range sg.Raised {
					if v {
						raised++
					}
				}
				shares = append(shares, raised/float64(len(sg.Raised)))
			}
		}
		return shares
	}
	macro := func(suite string, numbers []string) float64 {
		var xs []float64
		for _, n := range numbers {
			xs = append(xs, text.Mean(perPR(n, suite)))
		}
		return text.Mean(xs)
	}
	spread := func(suite string) float64 {
		var gaps []float64
		for _, n := range common {
			xs := perPR(n, suite)
			if len(xs) > 1 {
				lo, hi := xs[0], xs[0]
				for _, x := range xs {
					lo, hi = min(lo, x), max(hi, x)
				}
				gaps = append(gaps, hi-lo)
			}
		}
		return text.Mean(gaps)
	}

	records, err := reviewRecords(results)
	if err != nil {
		return "", err
	}
	var held, dev []string
	for _, n := range common {
		if Dev[n] {
			dev = append(dev, n)
		} else {
			held = append(held, n)
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Held-out PRs: %s. Dev PRs: %s.\n\n", listPRs(held), listPRs(dev))
	b.WriteString("| suite | human points raised, held-out | dev | run-to-run spread | other findings per review: valid / invalid / unsure | cost per review |\n")
	b.WriteString("|---|---|---|---|---|---|\n")
	order := append([]string{}, suites...)
	sort.SliceStable(order, func(i, j int) bool { return macro(order[i], held) > macro(order[j], held) })
	for _, suite := range order {
		kinds := map[string][]float64{"valid": nil, "invalid": nil, "unsure": nil}
		for _, n := range common {
			for _, g := range graded[n] {
				sg, ok := g[suite]
				if !ok {
					continue
				}
				for kind := range kinds {
					var perCall []float64
					for _, call := range sg.OtherFindings {
						count := 0.0
						for _, f := range call {
							if f.Valid == kind {
								count++
							}
						}
						perCall = append(perCall, count)
					}
					kinds[kind] = append(kinds[kind], text.Mean(perCall))
				}
			}
		}
		var costs []float64
		for _, r := range records {
			if r.Skill == suite && r.CostUSD != nil {
				costs = append(costs, *r.CostUSD)
			}
		}
		fmt.Fprintf(&b, "| %s | %.0f%% | %.0f%% | ±%.0f%% | %.1f / %.1f / %.1f | $%.2f |\n",
			suite, 100*macro(suite, held), 100*macro(suite, dev), 100*spread(suite),
			text.Mean(kinds["valid"]), text.Mean(kinds["invalid"]), text.Mean(kinds["unsure"]), text.Mean(costs))
	}
	b.WriteString("\nEach PR's human points, and how many runs of each suite raised each (a split vote counts as the majority said):\n\n")
	b.WriteString("| PR | point | " + strings.Join(suites, " | ") + " |\n")
	b.WriteString("|---|---|" + strings.Repeat("---|", len(suites)) + "\n")
	split, total := 0, 0
	for _, n := range common {
		for _, p := range pts[n] {
			var cells []string
			for _, suite := range suites {
				runs, raised := 0, 0
				for _, g := range graded[n] {
					sg, ok := g[suite]
					if !ok {
						continue
					}
					runs++
					if sg.Raised[p.ID] {
						raised++
					}
					yes := 0
					for _, v := range sg.Votes[p.ID] {
						if v {
							yes++
						}
					}
					total++
					if yes > 0 && yes < len(sg.Votes[p.ID]) {
						split++
					}
				}
				cells = append(cells, fmt.Sprintf("%d/%d", raised, runs))
			}
			fmt.Fprintf(&b, "| #%s | %s | %s |\n", n, p.Point, strings.Join(cells, " | "))
		}
	}
	fmt.Fprintf(&b, "\nGrader votes split on %d of %d verdicts.\n", split, total)
	if len(noPoints) > 0 {
		fmt.Fprintf(&b, "PRs with no substantive human point, left out: %s.\n", listPRs(noPoints))
	}
	for _, r := range records {
		if !r.Clean() {
			fmt.Fprintf(&b, "Left out: #%s %s (%s).\n", r.PR, r.Skill, r.Why())
		}
	}
	return b.String(), nil
}

func reviewRecords(results string) ([]ReviewRecord, error) {
	files, err := filepath.Glob(filepath.Join(results, "*-r*.json"))
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	var out []ReviewRecord
	for _, path := range files {
		if strings.HasSuffix(path, ".grade.json") {
			continue
		}
		var r ReviewRecord
		if err := readJSON(path, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func listPRs(numbers []string) string {
	if len(numbers) == 0 {
		return "none"
	}
	tagged := make([]string, len(numbers))
	for i, n := range numbers {
		tagged[i] = "#" + n
	}
	return strings.Join(tagged, ", ")
}

func sortedKeys(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
