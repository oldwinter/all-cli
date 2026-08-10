package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/oldwinter/all-cli/internal/tools"
)

func registryForCategoriesFilter(reg []tools.ToolDefinition, categoriesFilter string) ([]tools.ToolDefinition, error) {
	if strings.TrimSpace(categoriesFilter) == "" {
		return reg, nil
	}

	categories := make(map[string]bool)
	for _, category := range strings.Split(categoriesFilter, ",") {
		category = strings.ToLower(strings.TrimSpace(category))
		if category != "" {
			categories[category] = true
		}
	}
	if len(categories) == 0 {
		return nil, fmt.Errorf("invalid --categories value")
	}

	knownCategories := make(map[string]bool)
	for _, def := range defaultRegistry() {
		knownCategories[strings.ToLower(def.Category)] = true
	}
	unknown := make([]string, 0)
	for category := range categories {
		if !knownCategories[category] {
			unknown = append(unknown, category)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return nil, fmt.Errorf("unknown categories: %s", strings.Join(unknown, ", "))
	}

	filtered := make([]tools.ToolDefinition, 0, len(reg))
	for _, def := range reg {
		if categories[strings.ToLower(def.Category)] {
			filtered = append(filtered, def)
		}
	}
	return filtered, nil
}
