package cli

import (
	"strings"

	"github.com/oldwinter/all-cli/internal/tools"
)

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
		if distance < bestDistance || (distance == bestDistance && (bestID == "" || def.ID < bestID)) {
			bestID = def.ID
			bestDistance = distance
		}
	}
	if bestDistance > maxDistance {
		return ""
	}
	return bestID
}
