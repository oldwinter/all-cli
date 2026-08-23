package repopolicy

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Limits struct {
	MaxBytes int64
	MaxLines int
}

type Violation struct {
	Rule    string
	Path    string
	Line    int
	Message string
}

var (
	debtMarkerPattern  = regexp.MustCompile(`\b(TODO|FIXME|HACK|XXX)\b`)
	issueLinkPattern   = regexp.MustCompile(`(?i)(#\d+|GH-\d+|https://github\.com/[^/\s]+/[^/\s]+/issues/\d+)`)
	justCommandPattern = regexp.MustCompile(
		"`just[ \t]+([A-Za-z_][A-Za-z0-9_-]*)",
	)
	justRecipePattern = regexp.MustCompile(
		`(?m)^([A-Za-z_][A-Za-z0-9_-]*)(?:[ \t]+[^:=\n]+)?[ \t]*:`,
	)
	markdownLinkPattern = regexp.MustCompile(`\[[^]]+\]\(([^)]+)\)`)
)

func Audit(root string, limits Limits) ([]Violation, error) {
	paths, err := repositoryFiles(root)
	if err != nil {
		return nil, err
	}

	var violations []Violation
	for _, path := range paths {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		info, err := os.Stat(fullPath)
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", path, err)
		}
		if !info.Mode().IsRegular() {
			continue
		}

		content, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if limits.MaxBytes > 0 && info.Size() > limits.MaxBytes {
			violations = append(violations, Violation{
				Rule:    "file-size",
				Path:    path,
				Message: fmt.Sprintf("%d bytes exceeds the %d-byte limit", info.Size(), limits.MaxBytes),
			})
		}
		if lines := lineCount(content); limits.MaxLines > 0 && lines > limits.MaxLines {
			violations = append(violations, Violation{
				Rule:    "file-lines",
				Path:    path,
				Message: fmt.Sprintf("%d lines exceeds the %d-line limit", lines, limits.MaxLines),
			})
		}

		debtViolations, err := auditDebtMarkers(path, content)
		if err != nil {
			return nil, err
		}
		violations = append(violations, debtViolations...)
	}

	sortViolations(violations)
	return violations, nil
}

func CheckAgentGuide(root string) ([]Violation, error) {
	guidePath := filepath.Join(root, "AGENTS.md")
	guide, err := os.ReadFile(guidePath)
	if err != nil {
		return nil, fmt.Errorf("read AGENTS.md: %w", err)
	}
	justfile, err := os.ReadFile(filepath.Join(root, "justfile"))
	if err != nil {
		return nil, fmt.Errorf("read justfile: %w", err)
	}

	recipes := make(map[string]struct{})
	for _, match := range justRecipePattern.FindAllSubmatch(justfile, -1) {
		recipes[string(match[1])] = struct{}{}
	}

	var violations []Violation
	for _, match := range justCommandPattern.FindAllSubmatchIndex(guide, -1) {
		recipe := string(guide[match[2]:match[3]])
		if _, ok := recipes[recipe]; ok {
			continue
		}
		violations = append(violations, Violation{
			Rule:    "agent-command",
			Path:    "AGENTS.md",
			Line:    lineAt(guide, match[0]),
			Message: fmt.Sprintf("references undefined just recipe %q", recipe),
		})
	}

	for _, match := range markdownLinkPattern.FindAllSubmatchIndex(guide, -1) {
		target := strings.TrimSpace(string(guide[match[2]:match[3]]))
		if !isLocalLink(target) {
			continue
		}
		target = strings.TrimPrefix(target, "<")
		target = strings.TrimSuffix(target, ">")
		target = strings.SplitN(target, "#", 2)[0]
		target = strings.SplitN(target, "?", 2)[0]
		if target == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(filepath.Dir(guidePath), filepath.FromSlash(target))); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("check AGENTS.md link %q: %w", target, err)
		}
		violations = append(violations, Violation{
			Rule:    "agent-link",
			Path:    "AGENTS.md",
			Line:    lineAt(guide, match[0]),
			Message: fmt.Sprintf("references missing local path %q", target),
		})
	}

	sortViolations(violations)
	return violations, nil
}

func repositoryFiles(root string) ([]string, error) {
	cmd := exec.Command("git", "-C", root, "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("list repository files: %w", err)
	}

	rawPaths := bytes.Split(out, []byte{0})
	paths := make([]string, 0, len(rawPaths))
	for _, rawPath := range rawPaths {
		if len(rawPath) == 0 {
			continue
		}
		path := filepath.ToSlash(string(rawPath))
		if excludedPath(path) {
			continue
		}
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func excludedPath(path string) bool {
	for _, prefix := range []string{"dist/", "qa-bin/", "vendor/", ".git/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func lineCount(content []byte) int {
	if len(content) == 0 {
		return 0
	}
	lines := bytes.Count(content, []byte{'\n'})
	if content[len(content)-1] != '\n' {
		lines++
	}
	return lines
}

func auditDebtMarkers(path string, content []byte) ([]Violation, error) {
	if filepath.Ext(path) == ".go" {
		return auditGoDebtMarkers(path, content)
	}
	if !isCommentBearingText(path) {
		return nil, nil
	}
	return debtViolationsForText(path, content, 1), nil
}

func auditGoDebtMarkers(path string, content []byte) ([]Violation, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse %s for debt markers: %w", path, err)
	}
	if bytes.Contains(content, []byte("Code generated")) {
		return nil, nil
	}

	var violations []Violation
	for _, group := range file.Comments {
		for _, comment := range group.List {
			startLine := fset.Position(comment.Pos()).Line
			violations = append(violations, debtViolationsForText(path, []byte(comment.Text), startLine)...)
		}
	}
	return violations, nil
}

func debtViolationsForText(path string, content []byte, startLine int) []Violation {
	var violations []Violation
	for offset, line := range strings.Split(string(content), "\n") {
		if !debtMarkerPattern.MatchString(line) || issueLinkPattern.MatchString(line) {
			continue
		}
		marker := debtMarkerPattern.FindString(line)
		violations = append(violations, Violation{
			Rule:    "debt-marker",
			Path:    path,
			Line:    startLine + offset,
			Message: fmt.Sprintf("%s must reference a GitHub issue", marker),
		})
	}
	return violations
}

func isCommentBearingText(path string) bool {
	switch filepath.Ext(path) {
	case ".bash", ".mk", ".sh", ".toml", ".yaml", ".yml":
		return true
	}
	base := filepath.Base(path)
	return base == "Dockerfile" || base == "justfile"
}

func lineAt(content []byte, index int) int {
	return bytes.Count(content[:index], []byte{'\n'}) + 1
}

func isLocalLink(target string) bool {
	lower := strings.ToLower(target)
	return target != "" &&
		!strings.HasPrefix(target, "#") &&
		!strings.HasPrefix(lower, "http://") &&
		!strings.HasPrefix(lower, "https://") &&
		!strings.HasPrefix(lower, "mailto:")
}

func sortViolations(violations []Violation) {
	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Path != violations[j].Path {
			return violations[i].Path < violations[j].Path
		}
		if violations[i].Rule != violations[j].Rule {
			return violations[i].Rule < violations[j].Rule
		}
		return violations[i].Line < violations[j].Line
	})
}
