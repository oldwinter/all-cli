# all-cli

`all-cli` is a small CLI tool that inspects whether common CLI tools are installed/configured, and (in v0.1) manages “contexts” for:

- `kubectl` (context + namespace)
- `docker` (docker contexts)
- `gh` (active account per host)
- `glab` (GitLab host context)

It also detects “current context” for other common tools in `all-cli status` (read-only), including `aws`, `aliyun`, `wrangler`, `argocd`, `kargo`, `mise`, and `k9s`.

It defaults to human-friendly output and supports `--json` for stable machine-readable output (e.g. a future SwiftUI macOS app).

## Install

### From source (local)

```bash
go build ./cmd/all-cli
./all-cli version
```

### go install

```bash
go install github.com/oldwinter/all-cli/cmd/all-cli@latest
all-cli version
```

## Usage

### Global overview

```bash
all-cli status
all-cli status --json
all-cli status --tools kubectl,docker
all-cli status --timeout 10s
```

### kubectl

```bash
all-cli kubectl status
all-cli kubectl current
all-cli kubectl list
all-cli kubectl use <context> --namespace <ns>
all-cli kubectl namespace <ns>
```

### docker

```bash
all-cli docker status
all-cli docker current
all-cli docker list
all-cli docker use <context>
```

### gh

```bash
all-cli gh status
all-cli gh list --json
all-cli gh use --hostname github.com --user <login>
```

### glab

```bash
all-cli glab status
all-cli glab list
all-cli glab use <host>
```

## Security notes

- `all-cli` does **not** print or read plaintext tokens/secrets.
- It avoids `--show-token` flags and only uses official CLI outputs to determine “configured” state.
