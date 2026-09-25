package cli

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/oldwinter/all-cli/internal/execx"
	"github.com/oldwinter/all-cli/internal/model"
)

const (
	doctorInstallerAuto = "auto"
	doctorInstallerBrew = "brew"
	doctorInstallerNPM  = "npm"
	doctorInstallerPipx = "pipx"
	doctorInstallerGo   = "go"

	// Package installs routinely outlive the status probe --timeout, so each
	// install command gets its own ceiling.
	doctorInstallTimeout = 10 * time.Minute
)

type doctorFixOptions struct {
	DryRun    bool
	Installer string
}

type doctorInstallCommand struct {
	Installer string
	Name      string
	Args      []string
}

var doctorLookPath = execx.LookPath

// Installers are listed in auto-selection preference order.
var doctorInstallCommandsByTool = map[string][]doctorInstallCommand{
	"fd":         {brewInstall("fd")},
	"rg":         {brewInstall("ripgrep")},
	"fzf":        {brewInstall("fzf"), goInstall("github.com/junegunn/fzf@latest")},
	"zoxide":     {brewInstall("zoxide")},
	"eza":        {brewInstall("eza")},
	"bat":        {brewInstall("bat")},
	"yq":         {brewInstall("yq"), goInstall("github.com/mikefarah/yq/v4@latest")},
	"mise":       {brewInstall("mise")},
	"uv":         {brewInstall("uv"), pipxInstall("uv")},
	"just":       {brewInstall("just")},
	"obsidian":   {brewInstall("--cask", "obsidian")},
	"yazi":       {brewInstall("yazi")},
	"k9s":        {brewInstall("k9s")},
	"lazydocker": {brewInstall("lazydocker")},
	"aws":        {brewInstall("awscli")},
	"aliyun":     {brewInstall("aliyun-cli")},
	"wrangler":   {npmInstall("wrangler"), brewInstall("cloudflare-wrangler")},
	"vercel":     {npmInstall("vercel"), brewInstall("vercel-cli")},
	"railway":    {brewInstall("railway")},
	"netlify":    {npmInstall("netlify-cli"), brewInstall("netlify-cli")},
	"eksctl":     {brewInstall("eksctl")},
	"kubectl":    {brewInstall("kubernetes-cli")},
	"kubectx":    {brewInstall("kubectx")},
	"kubens":     {brewInstall("kubectx")},
	"kubecolor":  {brewInstall("kubecolor")},
	"krew":       {brewInstall("krew")},
	"kubefwd":    {brewInstall("txn2/tap/kubefwd")},
	"kubeshark":  {brewInstall("kubeshark")},
	"docker":     {brewInstall("docker")},
	"gh":         {brewInstall("gh")},
	"glab":       {brewInstall("glab")},
	"claude":     {npmInstall("@anthropic-ai/claude-code")},
	"codex":      {npmInstall("@openai/codex")},
	"openclaw":   {npmInstall("openclaw")},
	"opencode":   {brewInstall("anomalyco/tap/opencode"), npmInstall("opencode-ai")},
	"gemini":     {npmInstall("@google/gemini-cli")},
	"ccusage":    {npmInstall("ccusage")},
	"opencli":    {npmInstall("@jackwener/opencli")},
	"rclone":     {brewInstall("rclone")},
	"kargo":      {brewInstall("akuity/tap/kargo")},
	"argocd":     {brewInstall("argocd")},
}

func brewInstall(args ...string) doctorInstallCommand {
	return doctorInstallCommand{Installer: doctorInstallerBrew, Name: "brew", Args: append([]string{"install"}, args...)}
}

func npmInstall(pkg string) doctorInstallCommand {
	return doctorInstallCommand{Installer: doctorInstallerNPM, Name: "npm", Args: []string{"install", "-g", pkg}}
}

func pipxInstall(pkg string) doctorInstallCommand {
	return doctorInstallCommand{Installer: doctorInstallerPipx, Name: "pipx", Args: []string{"install", pkg}}
}

func goInstall(pkg string) doctorInstallCommand {
	return doctorInstallCommand{Installer: doctorInstallerGo, Name: "go", Args: []string{"install", pkg}}
}

func (c doctorInstallCommand) Command() []string {
	return append([]string{c.Name}, c.Args...)
}

func normalizeDoctorInstaller(installer string) (string, error) {
	installer = strings.ToLower(strings.TrimSpace(installer))
	switch installer {
	case "":
		return doctorInstallerAuto, nil
	case doctorInstallerAuto, doctorInstallerBrew, doctorInstallerNPM, doctorInstallerPipx, doctorInstallerGo:
		return installer, nil
	default:
		return "", fmt.Errorf("invalid --installer value %q (allowed: auto, brew, npm, pipx, go)", installer)
	}
}

// runDoctorFixes expects opts.Installer to be normalized already.
func runDoctorFixes(ctx context.Context, runner execx.Runner, report model.DiagnosticReport, opts doctorFixOptions) model.DoctorFixRun {
	out := model.DoctorFixRun{
		DryRun:    opts.DryRun,
		Installer: opts.Installer,
		Items:     []model.DoctorFixItem{},
	}
	seen := map[string]bool{}
	for _, diagnostic := range report.Diagnostics {
		toolID := strings.TrimSpace(diagnostic.RelatedTool)
		if !isInstallDiagnostic(diagnostic) || toolID == "" || seen[toolID] {
			continue
		}
		seen[toolID] = true
		out.Items = append(out.Items, runDoctorFix(ctx, runner, toolID, opts))
	}
	out.Summary = summarizeDoctorFixes(out.Items)
	return out
}

func runDoctorFix(ctx context.Context, runner execx.Runner, toolID string, opts doctorFixOptions) model.DoctorFixItem {
	item := model.DoctorFixItem{ToolID: toolID}
	cmd, reason := selectDoctorInstallCommand(toolID, opts.Installer)
	if cmd.Name == "" {
		item.Status = model.DoctorFixSkipped
		item.Reason = reason
		return item
	}
	item.Installer = cmd.Installer
	item.Command = cmd.Command()
	item.Supported = true
	if reason != "" {
		item.Status = model.DoctorFixSkipped
		item.Reason = reason
		return item
	}
	if opts.DryRun {
		item.Status = model.DoctorFixDryRun
		return item
	}
	res := runner.Run(ctx, cmd.Name, cmd.Args...)
	item.ExitCode = res.ExitCode
	if res.OK() {
		item.Status = model.DoctorFixInstalled
		return item
	}
	item.Status = model.DoctorFixFailed
	item.Reason = doctorCommandFailure(res)
	return item
}

// selectDoctorInstallCommand returns an empty command when the tool has no
// install recipe for the requested installer, and a command plus a skip reason
// when the recipe exists but its installer binary is not in PATH.
func selectDoctorInstallCommand(toolID, installer string) (doctorInstallCommand, string) {
	commands := doctorInstallCommandsByTool[toolID]
	if len(commands) == 0 {
		return doctorInstallCommand{}, "no supported automatic installer for this tool"
	}
	if installer != doctorInstallerAuto {
		for _, cmd := range commands {
			if cmd.Installer != installer {
				continue
			}
			if _, err := doctorLookPath(cmd.Name); err != nil {
				return cmd, fmt.Sprintf("installer %s was not found in PATH", cmd.Name)
			}
			return cmd, ""
		}
		return doctorInstallCommand{}, fmt.Sprintf("%s installer is not supported for this tool", installer)
	}

	tried := make([]string, 0, len(commands))
	for _, cmd := range commands {
		if _, err := doctorLookPath(cmd.Name); err == nil {
			return cmd, ""
		}
		tried = append(tried, cmd.Name)
	}
	return commands[0], fmt.Sprintf("no supported installer found in PATH (tried: %s)", strings.Join(tried, ", "))
}

func summarizeDoctorFixes(items []model.DoctorFixItem) model.DoctorFixSummary {
	summary := model.DoctorFixSummary{Total: len(items)}
	for _, item := range items {
		if item.Supported {
			summary.Supported++
		}
		switch item.Status {
		case model.DoctorFixInstalled:
			summary.Installed++
		case model.DoctorFixDryRun:
			summary.DryRun++
		case model.DoctorFixFailed:
			summary.Failed++
		default:
			summary.Skipped++
		}
	}
	return summary
}

func printDoctorFixes(w io.Writer, fixes model.DoctorFixRun) {
	fmt.Fprintln(w, "\nFixes")
	fmt.Fprintf(w, "  installer=%s dry_run=%t total=%d installed=%d preview=%d skipped=%d failed=%d\n",
		fixes.Installer,
		fixes.DryRun,
		fixes.Summary.Total,
		fixes.Summary.Installed,
		fixes.Summary.DryRun,
		fixes.Summary.Skipped,
		fixes.Summary.Failed,
	)
	if len(fixes.Items) == 0 {
		fmt.Fprintln(w, "  No missing tools to install.")
		return
	}
	for _, item := range fixes.Items {
		detail := item.Reason
		if len(item.Command) > 0 {
			detail = strings.Join(item.Command, " ")
			if item.Reason != "" {
				detail += " (" + item.Reason + ")"
			}
		}
		fmt.Fprintf(w, "  %-14s %-9s %s\n", item.ToolID, displayDoctorFixStatus(item.Status), detail)
	}
}

func displayDoctorFixStatus(status model.DoctorFixStatus) string {
	if status == model.DoctorFixDryRun {
		return "preview"
	}
	return string(status)
}

func doctorFixError(fixes model.DoctorFixRun) error {
	if fixes.Summary.Failed == 0 {
		return nil
	}
	return fmt.Errorf("%d install command(s) failed", fixes.Summary.Failed)
}

func isInstallDiagnostic(item model.DiagnosticItem) bool {
	return len(item.SuggestedActions) > 0 && item.SuggestedActions[0].ID == "install_tool"
}

func doctorCommandFailure(res execx.CmdResult) string {
	var parts []string
	if res.Err != nil {
		parts = append(parts, res.Err.Error())
	}
	if stderr := compactDoctorCommandOutput(res.Stderr); stderr != "" {
		parts = append(parts, stderr)
	}
	if len(parts) == 0 {
		if stdout := compactDoctorCommandOutput(res.Stdout); stdout != "" {
			parts = append(parts, stdout)
		}
	}
	if len(parts) == 0 {
		return fmt.Sprintf("command exited with code %d", res.ExitCode)
	}
	return strings.Join(parts, ": ")
}

func compactDoctorCommandOutput(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	const maxLen = 240
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
