# Release and rollback runbook

## Observe deployment impact

Every push to `main` creates a GitHub production deployment and a checksummed
release. Use:

- **Actions → release** for build/test/publish status,
- **Environments → production** for deployment history,
- the release summary for tag and asset verification,
- `SENTRY_DASHBOARD_URL` for errors by release,
- `PROMETHEUS_DASHBOARD_URL` for error ratio and duration,
- `POSTHOG_DASHBOARD_URL` for adoption of the new release.

Dashboard URLs are maintained as repository Actions variables, not committed
credentials.

## Failed release

1. Preserve the failed workflow link and step logs in a `type:bug`,
   `area:ci` issue.
2. Determine whether tagging, GitHub release publication, or Homebrew cask
   update occurred.
3. If nothing published, fix forward through a reviewed pull request.
4. If a partial release published, mark the GitHub release as a prerelease,
   explain the incomplete state, and stop Homebrew promotion.
5. Rerun only after `just check` and `goreleaser check` pass at the intended
   commit.

## Bad published release

1. Classify user impact and open a P0/P1 incident issue.
2. Do not rewrite or delete the existing tag; released checksums are immutable
   evidence.
3. Revert the faulty change through a protected pull request.
4. Publish the next release and verify all four platform archives plus
   `checksums.txt`.
5. Confirm error rate and command success recover on the dashboards.
6. Add release notes identifying the superseded version.

## Completion

Close the incident only after release assets, Homebrew installation, `all-cli
version`, and one representative `all-cli status --group-by none` smoke test
are verified.

When promoting the first release with `homebrew_casks`, migrate the Tap's old
`Formula/all-cli.rb` entry with `tap_migrations.json` and remove the obsolete
Formula in the same reviewed Tap change. Do not delete the Formula before the
new Cask has been published and installation has been verified.
