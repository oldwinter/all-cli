package cli

import (
	"strings"

	"github.com/oldwinter/all-cli/internal/tools"
)

func buildCatalogWordsReport(query string, registry []tools.ToolDefinition) catalogReport {
	report := buildCatalogReport("", registry)
	words := strings.Fields(strings.ToLower(query))
	matched := report.Tools[:0]
	for _, tool := range report.Tools {
		if catalogToolMatchesWords(tool, words) {
			matched = append(matched, tool)
		}
	}
	report.Query = query
	report.Tools = matched
	report.Count = len(matched)
	return report
}

func catalogToolMatchesWords(tool catalogTool, words []string) bool {
	for _, word := range words {
		if !catalogToolMatches(tool, word) {
			return false
		}
	}
	return true
}
