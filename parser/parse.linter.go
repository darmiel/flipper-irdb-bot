package parser

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"strconv"
	"strings"
)

type indicator struct {
	Start int
	End   int
}

type result struct {
	ExitRule   int `json:"exit_rule"`
	Indicators []indicator
	Error      string
	Suggestion string
}

type fatResult struct {
	Lnr    int
	Line   string
	Result result
}

///

type LinterParser struct {
	PythonPath string
	LinterRoot string
}

func (l *LinterParser) Raw(filePath string) ([]byte, error) {
	// run duplicate checker
	cmd := exec.Command(
		l.PythonPath,
		path.Join(l.LinterRoot, "main.py"),
		"json",
		filePath,
	)
	cmd.Env = []string{
		"PYTHONPATH=" + l.LinterRoot,
	}
	return cmd.Output()
}

func (l *LinterParser) Pretty(filePath string) ([]fatResult, string, error) {
	data, err := l.Raw(filePath)
	if err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			return nil, "", err
		}
	}

	var files map[string][]fatResult
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, "", errors.New("cannot parse linter result")
	}
	var results []fatResult
	for _, res := range files {
		results = res
		break
	}

	var bob strings.Builder
	for _, result := range results {
		if bob.Len() > 0 {
			bob.WriteString("\n")
		}

		bob.WriteString(":lady_beetle: Found an issue in line " + strconv.Itoa(result.Lnr) + ":\n")

		maxEnd := 0
		for _, indicator := range result.Result.Indicators {
			if maxEnd < indicator.End {
				maxEnd = indicator.End
			}
		}
		indicatorStr := strings.Repeat(" ", maxEnd)
		for _, indicator := range result.Result.Indicators {
			indicatorStr = indicatorStr[:indicator.Start] +
				strings.Repeat("^", indicator.End-indicator.Start) +
				indicatorStr[indicator.End:]
		}

		bob.WriteString(fmt.Sprintf("```\n%s\n%s\n%s\n```\n", result.Line, indicatorStr, result.Result.Error))
		if result.Result.Suggestion != "" {
			bob.WriteString(fmt.Sprintf("> **suggested:** %s\n", result.Result.Suggestion))
		}
	}
	return results, bob.String(), nil
}
