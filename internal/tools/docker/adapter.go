package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"github.com/oldwinter/all-cli/internal/execx"
)

type Adapter struct {
	runner execx.Runner
}

func New(runner execx.Runner) Adapter {
	return Adapter{runner: runner}
}

type Context struct {
	Name        string `json:"name"`
	IsCurrent   bool   `json:"is_current"`
	Description string `json:"description,omitempty"`
}

type ContainerImage struct {
	ID     string `json:"id,omitempty"`
	Names  string `json:"names,omitempty"`
	Image  string `json:"image"`
	Status string `json:"status,omitempty"`
}

type UpdatePlanOptions struct {
	All    bool
	Images []string
}

type ImageUpdate struct {
	Image            string   `json:"image"`
	Command          []string `json:"command"`
	SourceContainers []string `json:"source_containers,omitempty"`
	Applied          bool     `json:"applied,omitempty"`
	Error            string   `json:"error,omitempty"`
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	contexts, warnings, errs, err := a.ListContexts(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return len(contexts) > 0, warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	res := a.runner.Run(ctx, "docker", "context", "show")
	if res.Err != nil {
		cause := execx.ErrMessage(res)
		if cause == "" {
			return nil, nil, nil, fmt.Errorf("docker context show failed (exit=%d)", res.ExitCode)
		}
		return nil, nil, []string{cause}, fmt.Errorf("docker context show failed (exit=%d): %s", res.ExitCode, cause)
	}
	cur := map[string]string{}
	if v := strings.TrimSpace(res.Stdout); v != "" {
		cur["context"] = v
	}
	return cur, nil, nil, nil
}

func (a Adapter) ListContexts(ctx context.Context) ([]Context, []string, []string, error) {
	res := a.runner.Run(ctx, "docker", "context", "ls", "--format", "{{json .}}")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return nil, nil, []string{errMsg}, fmt.Errorf("docker context ls failed (exit=%d)", res.ExitCode)
	}
	return parseContextLSJSONLines(res.Stdout)
}

func (a Adapter) UseContext(ctx context.Context, contextName string) error {
	res := a.runner.Run(ctx, "docker", "context", "use", contextName)
	if res.Err != nil {
		cause := execx.ErrMessage(res)
		if cause == "" {
			return fmt.Errorf("docker context use %q failed (exit=%d)", contextName, res.ExitCode)
		}
		return fmt.Errorf("docker context use %q failed (exit=%d): %s", contextName, res.ExitCode, cause)
	}
	return nil
}

func (a Adapter) ListContainerImages(ctx context.Context, all bool) ([]ContainerImage, []string, []string, error) {
	args := []string{"ps"}
	if all {
		args = append(args, "-a")
	}
	args = append(args, "--format", "{{json .}}")

	res := a.runner.Run(ctx, "docker", args...)
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return nil, nil, []string{errMsg}, fmt.Errorf("docker ps failed (exit=%d)", res.ExitCode)
	}
	return parsePSJSONLines(res.Stdout)
}

func (a Adapter) BuildUpdatePlan(ctx context.Context, opts UpdatePlanOptions) ([]ImageUpdate, []string, []string, error) {
	imagesByRef := map[string]map[string]bool{}
	warnings := []string{}
	errs := []string{}

	addImage := func(ref, source string) {
		image, reason, ok := normalizeImageRef(ref)
		if !ok {
			if strings.TrimSpace(ref) != "" {
				warnings = append(warnings, fmt.Sprintf("skipping image %q: %s", strings.TrimSpace(ref), reason))
			}
			return
		}
		if imagesByRef[image] == nil {
			imagesByRef[image] = map[string]bool{}
		}
		if strings.TrimSpace(source) != "" {
			imagesByRef[image][strings.TrimSpace(source)] = true
		}
	}

	if len(opts.Images) > 0 {
		for _, image := range opts.Images {
			addImage(image, "")
		}
	} else {
		containers, moreWarnings, moreErrs, err := a.ListContainerImages(ctx, opts.All)
		warnings = append(warnings, moreWarnings...)
		errs = append(errs, moreErrs...)
		if err != nil {
			return nil, warnings, errs, err
		}
		for _, container := range containers {
			addImage(container.Image, container.Names)
		}
	}

	images := make([]string, 0, len(imagesByRef))
	for image := range imagesByRef {
		images = append(images, image)
	}
	sort.Strings(images)

	updates := make([]ImageUpdate, 0, len(images))
	for _, image := range images {
		containers := make([]string, 0, len(imagesByRef[image]))
		for name := range imagesByRef[image] {
			containers = append(containers, name)
		}
		sort.Strings(containers)
		updates = append(updates, ImageUpdate{
			Image:            image,
			Command:          []string{"docker", "pull", image},
			SourceContainers: containers,
		})
	}
	if len(updates) == 0 {
		warnings = append(warnings, "no Docker image update candidates found")
	}
	return updates, warnings, errs, nil
}

func (a Adapter) PullImage(ctx context.Context, image string) error {
	res := a.runner.Run(ctx, "docker", "pull", image)
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return fmt.Errorf("docker pull %q failed (exit=%d): %s", image, res.ExitCode, errMsg)
	}
	return nil
}

type dockerContextLSItem struct {
	Current     bool   `json:"Current"`
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Error       string `json:"Error"`
}

func parseContextLSJSONLines(stdout string) ([]Context, []string, []string, error) {
	warnings := []string{}
	errs := []string{}
	var out []Context

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item dockerContextLSItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			errs = append(errs, fmt.Sprintf("failed to parse docker context JSON line: %v", err))
			continue
		}
		if strings.TrimSpace(item.Error) != "" {
			warnings = append(warnings, fmt.Sprintf("docker context %q error: %s", item.Name, strings.TrimSpace(item.Error)))
		}
		out = append(out, Context{
			Name:        item.Name,
			IsCurrent:   item.Current,
			Description: item.Description,
		})
	}
	if err := scanner.Err(); err != nil {
		return out, warnings, append(errs, err.Error()), err
	}
	return out, warnings, errs, nil
}

type dockerPSItem struct {
	ID     string `json:"ID"`
	Names  string `json:"Names"`
	Image  string `json:"Image"`
	Status string `json:"Status"`
}

func parsePSJSONLines(stdout string) ([]ContainerImage, []string, []string, error) {
	warnings := []string{}
	errs := []string{}
	var out []ContainerImage

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var item dockerPSItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			errs = append(errs, fmt.Sprintf("failed to parse docker ps JSON line: %v", err))
			continue
		}
		if strings.TrimSpace(item.Image) == "" {
			warnings = append(warnings, fmt.Sprintf("docker container %q has no image reference", strings.TrimSpace(item.Names)))
			continue
		}
		out = append(out, ContainerImage(item))
	}
	if err := scanner.Err(); err != nil {
		return out, warnings, append(errs, err.Error()), err
	}
	return out, warnings, errs, nil
}

func normalizeImageRef(ref string) (string, string, bool) {
	ref = strings.TrimSpace(ref)
	switch {
	case ref == "":
		return "", "empty image reference", false
	case ref == "<none>" || strings.Contains(ref, "<none>"):
		return "", "untagged image reference", false
	case strings.ContainsAny(ref, " \t\r\n"):
		return "", "image reference contains whitespace", false
	case strings.HasPrefix(ref, "sha256:") || looksLikeImageID(ref):
		return "", "image ID cannot be refreshed by tag", false
	case strings.Contains(ref, "@sha256:"):
		return "", "digest-pinned image reference cannot be refreshed by tag", false
	default:
		return ref, "", true
	}
}

func looksLikeImageID(ref string) bool {
	if len(ref) < 12 {
		return false
	}
	for _, r := range ref {
		if !unicode.IsDigit(r) && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}
