package parser

import (
	"encoding/json"
	"fmt"
	"math"
	"os/exec"
	"path"
	"strings"
)

type FlipperScriptsParser struct {
	PythonPath         string
	FlipperScriptsRoot string
}

type dupeBalance struct {
	Confidence    float32
	Adds          int
	AddsPercent   float32
	Misses        int
	MissesPercent float32
}

type dupeResult struct {
	Path       string
	Confidence float32
	Balance    dupeBalance
	Common     map[string][]string
}

func (f *FlipperScriptsParser) RawDupeCheck(filePath string) ([]byte, error) {
	// run duplicate checker
	cmd := exec.Command(
		f.PythonPath,
		path.Join(f.FlipperScriptsRoot, "infrared", "duplicate_finder_json.py"),
		filePath,
	)
	cmd.Env = []string{
		"PYTHONPATH=" + f.FlipperScriptsRoot,
	}
	return cmd.Output()
}

func createProgressBar(width int, percentage float32) string {
	filled := int(math.Ceil(float64(width) * float64(percentage)))
	var bob strings.Builder
	bob.WriteString("[")
	bob.WriteString(strings.Repeat("#", filled))
	bob.WriteString(strings.Repeat(" ", width-filled))
	bob.WriteString("]")
	return bob.String()
}

func (f *FlipperScriptsParser) PrettyDupeCheck(filePath string) ([]dupeResult, string, error) {
	data, err := f.RawDupeCheck(filePath)
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, "", err
		}
	}
	if len(strings.TrimSpace(string(data))) <= 0 {
		return nil, "", nil
	}
	// parse output
	var results map[string][]dupeResult
	if err = json.Unmarshal(data, &results); err != nil {
		return nil, "", err
	}

	var result []dupeResult
	for _, v := range results {
		result = v
		break
	}

	var bob strings.Builder
	for _, res := range result {
		if bob.Len() > 0 {
			bob.WriteString("\n")
		}

		stripPath := res.Path
		if strings.Contains(stripPath, "Flipper-IRDB") {
			stripPath = stripPath[strings.Index(stripPath, "Flipper-IRDB")+12:]
		}
		stripPath = strings.ReplaceAll(stripPath, "_", "\\_")

		bob.WriteString("**" + stripPath + "** (")
		bob.WriteString(fmt.Sprintf("%.2f%%", res.Confidence*100))
		bob.WriteString("):\n")
		bob.WriteString("<:confidence:1065025041164668949> `")
		bob.WriteString(createProgressBar(15, res.Confidence))
		bob.WriteString(fmt.Sprintf(" %.2f%%", res.Confidence*100.0))
		bob.WriteString("`\n<:balance:1065025039470170194> `")
		bob.WriteString(createProgressBar(15, res.Balance.Confidence))
		bob.WriteString(fmt.Sprintf(" %.2f%%", res.Balance.Confidence*100.0))
		bob.WriteString("`\n")
	}

	return result, bob.String(), nil
}
