package kubectl

import (
	"context"
	"fmt"
	"strings"

	"github.com/oldwinter/all-cli/internal/execx"
)

type Adapter struct {
	runner execx.Runner
}

func New(runner execx.Runner) Adapter {
	return Adapter{runner: runner}
}

func (a Adapter) Configured(ctx context.Context) (bool, []string, []string, error) {
	contexts, warnings, errs, err := a.ListContexts(ctx)
	if err != nil {
		return false, warnings, errs, err
	}
	return len(contexts) > 0, warnings, errs, nil
}

func (a Adapter) Current(ctx context.Context) (map[string]string, []string, []string, error) {
	warnings := []string{}
	errs := []string{}

	ctxName, err := a.currentContext(ctx)
	if err != nil {
		errs = append(errs, err.Error())
	}
	ns, err := a.currentNamespace(ctx)
	if err != nil {
		errs = append(errs, err.Error())
	}

	cur := map[string]string{}
	if strings.TrimSpace(ctxName) != "" {
		cur["context"] = ctxName
	}
	if strings.TrimSpace(ns) != "" {
		cur["namespace"] = ns
	}
	return cur, warnings, errs, nil
}

func (a Adapter) ListContexts(ctx context.Context) ([]string, []string, []string, error) {
	res := a.runner.Run(ctx, "kubectl", "config", "get-contexts", "-o", "name")
	if res.Err != nil {
		errMsg := strings.TrimSpace(res.Stderr)
		if errMsg == "" {
			errMsg = strings.TrimSpace(res.Stdout)
		}
		if errMsg == "" {
			errMsg = res.Err.Error()
		}
		return nil, nil, []string{errMsg}, fmt.Errorf("kubectl config get-contexts failed (exit=%d)", res.ExitCode)
	}

	var out []string
	for _, line := range strings.Split(res.Stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out, nil, nil, nil
}

func (a Adapter) UseContext(ctx context.Context, contextName string) error {
	res := a.runner.Run(ctx, "kubectl", "config", "use-context", contextName)
	if res.Err != nil {
		return fmt.Errorf("kubectl config use-context %q failed (exit=%d): %s", contextName, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

func (a Adapter) SetNamespaceForContext(ctx context.Context, contextName, namespace string) error {
	res := a.runner.Run(ctx, "kubectl", "config", "set-context", contextName, "--namespace", namespace)
	if res.Err != nil {
		return fmt.Errorf("kubectl config set-context %q --namespace %q failed (exit=%d): %s", contextName, namespace, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

func (a Adapter) SetNamespaceForCurrentContext(ctx context.Context, namespace string) error {
	res := a.runner.Run(ctx, "kubectl", "config", "set-context", "--current", "--namespace", namespace)
	if res.Err != nil {
		return fmt.Errorf("kubectl config set-context --current --namespace %q failed (exit=%d): %s", namespace, res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return nil
}

func (a Adapter) currentContext(ctx context.Context) (string, error) {
	res := a.runner.Run(ctx, "kubectl", "config", "current-context")
	if res.Err != nil {
		return "", fmt.Errorf("kubectl config current-context failed (exit=%d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return strings.TrimSpace(res.Stdout), nil
}

func (a Adapter) currentNamespace(ctx context.Context) (string, error) {
	res := a.runner.Run(ctx, "kubectl", "config", "view", "--minify", "--output", "jsonpath={..namespace}{\"\\n\"}")
	if res.Err != nil {
		return "", fmt.Errorf("kubectl config view (namespace) failed (exit=%d): %s", res.ExitCode, strings.TrimSpace(res.Stderr))
	}
	return strings.TrimSpace(res.Stdout), nil
}
