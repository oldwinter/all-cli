package tools

import "github.com/oldwinter/all-cli/internal/model"

func MetadataForTool(id string) model.ToolMetadata {
	switch id {
	case "fd":
		return naMetadata("Fast file finder for local filesystem search.")
	case "rg":
		return naMetadata("Fast text search tool for recursive code and file content search.")
	case "fzf":
		return naMetadata("Interactive fuzzy finder used to select items from terminal input.")
	case "zoxide":
		return naMetadata("Smart directory jumper that tracks and ranks frequently used paths.")
	case "eza":
		return naMetadata("Modern replacement for ls with richer file listing output.")
	case "bat":
		return naMetadata("File viewer with syntax highlighting and line numbers.")
	case "yq":
		return naMetadata("Command-line processor for YAML and structured data transformations.")
	case "brew":
		return naMetadata("Homebrew package manager for installing and updating local tooling.")
	case "uv":
		return naMetadata("Python-oriented package and tool runner used for fast environment and CLI workflows.")
	case "just":
		return naMetadata("Task runner for project automation recipes declared in a justfile.")
	case "obsidian":
		return naMetadata(
			"Obsidian desktop application for local markdown knowledge bases.",
			"The detected binary launches a desktop app, not a pure command-line workflow.",
		)
	case "yazi":
		return naMetadata("Terminal file manager for navigating and manipulating local files.")
	case "k9s":
		return currentMetadata(
			"Terminal dashboard for inspecting Kubernetes resources through the current kubeconfig context.",
			"Configuration is not evaluated directly; current reports the k9s config file and resolved Kubernetes context when available.",
			map[string]string{
				"config":    "Path to the k9s configuration file in use.",
				"context":   "The Kubernetes context k9s resolves for the current session.",
				"namespace": "The namespace k9s would open by default, if explicitly configured.",
			},
			[]string{"inspect_status", "show_current"},
			"Missing namespace is often a normal state because kubeconfig may not pin a default namespace.",
		)
	case "lazydocker":
		return naMetadata("Terminal dashboard for viewing and operating Docker resources.")
	case "aws":
		return currentMetadata(
			"Amazon Web Services CLI for account, infrastructure, and service operations.",
			"At least one AWS profile is discoverable through local AWS CLI configuration.",
			map[string]string{
				"profile": "The active AWS profile resolved from environment variables or local config.",
				"region":  "The active AWS region resolved from environment variables or local config.",
				"output":  "The default AWS CLI output format, if configured.",
			},
			[]string{"inspect_status", "show_current"},
		)
	case "aliyun":
		return currentMetadata(
			"Alibaba Cloud CLI for account and infrastructure operations.",
			"At least one Aliyun profile is available in local CLI configuration.",
			map[string]string{
				"profile":  "The current or preferred Aliyun profile name.",
				"region":   "The default region associated with the selected profile.",
				"language": "The CLI language preference for the selected profile.",
				"valid":    "Validation status reported by the Aliyun CLI for the selected profile.",
			},
			[]string{"inspect_status", "show_current"},
		)
	case "wrangler":
		return currentMetadata(
			"Cloudflare Wrangler CLI for Workers, Pages, and related Cloudflare resources.",
			"The current user is authenticated with Wrangler.",
			map[string]string{
				"logged_in":      "Whether Wrangler reports an authenticated user session.",
				"accounts_count": "How many Cloudflare accounts are visible to the authenticated user.",
				"account_id":     "The single visible account ID when exactly one account is available.",
			},
			[]string{"inspect_status", "show_current"},
			"Multiple accounts are not an error, but they mean there is no single globally implied target account.",
		)
	case "eksctl":
		return naMetadata("CLI for managing Amazon EKS clusters and related resources.")
	case "kubectl":
		return currentMetadata(
			"Kubernetes CLI used to inspect and operate clusters through kubeconfig contexts.",
			"At least one kubeconfig context is available and readable.",
			map[string]string{
				"context":   "The active kubeconfig context name.",
				"namespace": "The default namespace bound to the active context, if one is set.",
			},
			[]string{"inspect_status", "show_current", "list_contexts", "switch_context", "switch_namespace"},
			"Missing namespace is not always an error because many kubeconfig contexts rely on the cluster default namespace.",
		)
	case "kubectx":
		return naMetadata("Helper CLI for switching Kubernetes contexts quickly.")
	case "kubens":
		return naMetadata("Helper CLI for switching Kubernetes namespaces quickly.")
	case "kubecolor":
		return naMetadata("kubectl-compatible wrapper that adds colorized output.")
	case "krew":
		return naMetadata("Plugin manager for kubectl extensions.")
	case "kubefwd":
		return naMetadata("Utility for forwarding multiple Kubernetes services to localhost for local debugging.")
	case "kubeshark":
		return naMetadata("Kubernetes network inspection tool for observing in-cluster traffic.")
	case "docker":
		return currentMetadata(
			"Docker CLI for containers, images, networks, and Docker contexts.",
			"At least one Docker context is available and the Docker CLI config is readable.",
			map[string]string{
				"context": "The active Docker context name.",
			},
			[]string{"inspect_status", "show_current", "list_contexts", "switch_context"},
		)
	case "gh":
		return currentMetadata(
			"GitHub CLI for repository, issue, pull request, and workflow operations.",
			"At least one authenticated GitHub host/account is available to the CLI.",
			map[string]string{
				"hostname": "The GitHub hostname associated with the effective account selection.",
				"user":     "The login of the effective authenticated GitHub user.",
			},
			[]string{"inspect_status", "show_current", "list_accounts", "switch_account"},
		)
	case "glab":
		return currentMetadata(
			"GitLab CLI for repository, merge request, issue, and workflow operations.",
			"At least one usable GitLab instance is authenticated in local glab configuration.",
			map[string]string{
				"effective_host": "The GitLab host glab currently resolves for API operations in this environment.",
				"global_host":    "The global default GitLab host configured in glab.",
				"user":           "The effective authenticated GitLab user for the current host.",
			},
			[]string{"inspect_status", "show_current", "list_instances", "switch_host"},
			"`effective_host` and `global_host` may differ when repository or environment settings override the global default.",
		)
	case "linear":
		return naMetadata("Linear CLI for issue, project, and workflow management.")
	case "claude":
		return naMetadata("Claude Code CLI for AI-assisted coding and terminal workflows.")
	case "codex":
		return naMetadata("Codex CLI for AI-assisted coding and terminal workflows.")
	case "openclaw":
		return naMetadata("AI coding CLI for terminal-based development workflows.")
	case "opencode":
		return naMetadata("AI coding CLI for interactive code and terminal workflows.")
	case "gemini":
		return naMetadata("Gemini CLI for AI-assisted terminal and content workflows.")
	case "ccusage":
		return naMetadata("CLI for inspecting Claude Code usage and related local usage data.")
	case "litellm-proxy":
		return naMetadata("LiteLLM proxy CLI for local or staging model routing and smoke tests.")
	case "simplex-cli":
		return naMetadata("Internal operations CLI for Simplex product and account workflows.")
	case "rclone":
		return currentMetadata(
			"File sync and transfer CLI for local and remote storage backends.",
			"A readable rclone config file exists on disk.",
			nil,
			[]string{"inspect_status"},
		)
	case "kargo":
		return currentMetadata(
			"Kargo CLI for GitOps delivery and default project selection.",
			"A non-empty Kargo API address is configured locally.",
			map[string]string{
				"api_address": "The Kargo API endpoint configured for the current CLI context.",
				"project":     "The default Kargo project selected for CLI commands, if one is set.",
			},
			[]string{"inspect_status", "show_current", "switch_project"},
		)
	case "argocd":
		return currentMetadata(
			"Argo CD CLI for GitOps application management and server context switching.",
			"At least one Argo CD context is configured locally.",
			map[string]string{
				"context": "The active Argo CD context name.",
				"server":  "The Argo CD server address associated with the active context.",
			},
			[]string{"inspect_status", "show_current", "list_contexts", "switch_context"},
		)
	case "opensearch":
		return naMetadata("OpenSearch CLI for OpenSearch cluster and index operations.")
	case "mise":
		return currentMetadata(
			"Runtime manager that resolves active tool versions for the current shell and project.",
			"Configuration is not evaluated directly; current reports resolved runtime versions when available.",
			map[string]string{
				"*": "Each key is a runtime or tool name and each value is the resolved active version.",
			},
			[]string{"inspect_status", "show_current"},
		)
	default:
		return model.ToolMetadata{
			Purpose:        "Command-line tool tracked by all-cli.",
			ConfiguredWhen: "Configuration semantics are tool-specific and may not be available in this build.",
			AgentActions:   []string{"inspect_status"},
		}
	}
}

func naMetadata(purpose string, notes ...string) model.ToolMetadata {
	return model.ToolMetadata{
		Purpose:        purpose,
		ConfiguredWhen: "all-cli does not evaluate configuration for this tool; configured_state is reported as n/a when the binary is installed.",
		AgentActions:   []string{"inspect_status"},
		Notes:          notes,
	}
}

func currentMetadata(purpose, configuredWhen string, fields map[string]string, actions []string, notes ...string) model.ToolMetadata {
	return model.ToolMetadata{
		Purpose:                  purpose,
		ConfiguredWhen:           configuredWhen,
		CurrentFieldDescriptions: fields,
		AgentActions:             actions,
		Notes:                    notes,
	}
}
