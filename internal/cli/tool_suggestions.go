package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oldwinter/all-cli/internal/tools"
)

func unknownToolIDError(ids []string) error {
	orderedIDs := append([]string(nil), ids...)
	sort.Strings(orderedIDs)

	quotedIDs := make([]string, 0, len(orderedIDs))
	suggestions := make([]string, 0, len(orderedIDs))
	singularSuggestion := ""
	for _, id := range orderedIDs {
		quotedIDs = append(quotedIDs, fmt.Sprintf("%q", id))
		if suggestion := toolIDSuggestion(id); suggestion != "" {
			suggestions = append(suggestions, fmt.Sprintf("%q -> %q", id, suggestion))
			singularSuggestion = suggestion
		}
	}

	if len(orderedIDs) == 1 {
		if singularSuggestion != "" {
			return fmt.Errorf("unknown tool ID %s; did you mean %q?", quotedIDs[0], singularSuggestion)
		}
		return fmt.Errorf("unknown tool ID %s", quotedIDs[0])
	}
	if len(suggestions) > 0 {
		return fmt.Errorf("unknown tool IDs: %s; suggestions: %s", strings.Join(quotedIDs, ", "), strings.Join(suggestions, ", "))
	}
	return fmt.Errorf("unknown tool IDs: %s", strings.Join(quotedIDs, ", "))
}

func toolIDSuggestion(input string) string {
	inputRunes := []rune(strings.ToLower(strings.TrimSpace(input)))
	if len(inputRunes) == 0 {
		return ""
	}

	maxDistance := 2
	if len(inputRunes) <= 3 {
		maxDistance = 1
	} else if len(inputRunes) >= 8 {
		maxDistance = 3
	}

	bestID := ""
	bestDistance := maxDistance + 1
	bestMatches := 0
	for _, def := range tools.DefaultRegistry() {
		candidateRunes := []rune(def.ID)
		previous := make([]int, len(candidateRunes)+1)
		current := make([]int, len(candidateRunes)+1)
		for i := range previous {
			previous[i] = i
		}
		for i, inputRune := range inputRunes {
			current[0] = i + 1
			for j, candidateRune := range candidateRunes {
				cost := 0
				if inputRune != candidateRune {
					cost = 1
				}
				current[j+1] = min(previous[j+1]+1, current[j]+1, previous[j]+cost)
			}
			previous, current = current, previous
		}
		distance := previous[len(candidateRunes)]
		if distance < bestDistance {
			bestID = def.ID
			bestDistance = distance
			bestMatches = 1
		} else if distance == bestDistance {
			bestMatches++
		}
	}
	if bestDistance > maxDistance || bestMatches != 1 {
		return ""
	}
	return bestID
}
