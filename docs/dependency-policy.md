# Dependency update policy

Dependabot opens grouped weekly pull requests for Go modules, GitHub Actions,
and the development container.

## Minimum release age

Normal dependency releases must be at least **seven full days old** before a
version-update pull request is merged. The waiting period begins at the
upstream registry or release timestamp, not when Dependabot opens the pull
request.

The reviewer records the upstream release timestamp and earliest eligible
merge time in the pull request. If the release is younger than seven days,
leave the pull request open and allow CI and community reports to accumulate.

## Required review

Before merge:

1. Read upstream release notes and linked migration/security notes.
2. Verify source repository, module path, checksum, and maintainer identity.
3. Run `go mod verify`, `just check`, CodeQL, and dependency review.
4. Inspect transitive changes with `go mod graph` and `go mod why`.
5. Confirm the pull request includes a rollback plan.

## Security exception

The seven-day wait may be waived only when a linked security advisory shows
that delaying is riskier than updating. The pull request must:

- link the advisory or CVE,
- explain exploitability in `all-cli`,
- receive explicit maintainer approval,
- pass all required security and quality checks,
- document the previous known-good version and rollback command.

The exception changes timing only; it never bypasses tests, review, or branch
protection.
