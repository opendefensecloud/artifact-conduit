# Contributing

## How To Provide Feedback

Please [raise an issue in Github](https://github.com/opendefensecloud/artifact-conduit/issues).

## Code of Conduct

See [Code of Conduct](./CODE_OF_CONDUCT.md).

## Community Meetings (monthly)

There are currently no community meetings. Please raise an issue to reach out.

## Contributor Meetings (twice monthly)

There are currently no public contributor meetings. Please raise an issue to reach out.

## Slack

There is currently no public Slack. Please raise an issue to reach out.

## Commit Convention

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification. Both PR titles and individual commit messages are validated in CI.

### Format

```
<type>(optional scope): <description>
```

### Allowed Types

| Type       | Purpose                                              |
| ---------- | ---------------------------------------------------- |
| `feat`     | A new feature                                        |
| `fix`      | A bug fix                                            |
| `docs`     | Documentation changes                                |
| `chore`    | Maintenance tasks (deps, CI config, etc.)            |
| `refactor` | Code changes that neither fix a bug nor add a feature |
| `test`     | Adding or updating tests                             |
| `ci`       | CI/CD pipeline changes                               |
| `perf`     | Performance improvements                             |
| `revert`   | Reverting a previous commit                          |

### Examples

```
feat: add OCI artifact signing support
fix(registry): handle missing manifest digest
docs: update contributing guidelines
chore(deps): update cosign to v2.5.0
refactor(transfer): extract blob streaming logic
```

### Breaking Changes

Append `!` after the type/scope to indicate a breaking change:

```
feat!: change artifact transfer API
refactor(api)!: rename TransferPolicy to SyncPolicy
```

## How To Contribute

We're always looking for contributors.

### Authoring PRs

* Documentation - something missing or unclear? Please submit a pull request according to our [docs contribution guide](./developer-guide/documentation-changes.md)!
* Code contribution - investigate a [good first issue](https://github.com/opendefensecloud/artifact-conduit/issues?q=is%3Aopen+is%3Aissue+label%3A%22good+first+issue%22), [high priority bugs](#triaging-bugs), or anything not assigned.
* You can work on an issue without being assigned.

#### Running Locally

To run ARC locally for development: [running locally](./developer-guide/developing-locally.md).

#### Dependencies

Dependencies increase the risk of security issues and have on-going maintenance costs.

The dependency must pass these test:

* A strong use case.
* It has an acceptable license (e.g. MIT).
* It is actively maintained.
* It has no security issues.

Example, should we add `fasttemplate`, [view the Snyk report](https://snyk.io/advisor/golang/github.com/valyala/fasttemplate):

| Test                                    | Outcome                             |
|-----------------------------------------|-------------------------------------|
| A strong use case.                      | ❌ Fail. We can use `text/template`. |
| It has an acceptable license (e.g. MIT) | ✅ Pass. MIT license.               |
| It is actively maintained.              | ❌ Fail. Project is inactive.        |
| It has no security issues.              | ✅ Pass. No known security issues.  |

No, we should not add that dependency.

#### Test Policy

Changes without either unit or e2e tests are unlikely to be accepted.
See [the pull request template](https://github.com/opendefensecloud/artifact-conduit/blob/main/.github/pull_request_template.md).

### Other Contributions

* [Reviewing PRs](#reviewing-prs)
* Responding to questions in the [Slack](#slack) channels
* Responding to questions in [Github Discussions](https://github.com/opendefensecloud/artifact-conduit/discussions)
* [Triaging new bugs](#triaging-bugs)

#### Reviewing PRs

Anybody can review a PR.
If you are in a [designated role](#roles), add yourself as an "Assignee" to a PR if you plan to lead the review.
If you are a Reviewer or below, then once you have approved a PR, request a review from one or more Approvers and above.

#### Timeliness

We encourage PR authors and reviewers to respond to change requests in a reasonable time frame.
If you're on vacation or will be unavailable, please let others know on the PR.

##### PR Author Timeliness

If a PR hasn't seen activity from the author for 10 business days, someone else may ask to take it over.
We suggest commenting on the original PR and tagging the author to check on their plans.
Maintainers can reassign PRs to new contributors if the original author doesn't respond with a plan.
For PRs that have been inactive for 3 months, the takeover process can happen immediately.
**IMPORTANT:** If a PR is taken over and uses any code from the previous PR, the original author *must* be credited using `Co-authored-by` on the commits.

#### Triaging Bugs

New bugs need to be triaged to identify the highest priority ones.
