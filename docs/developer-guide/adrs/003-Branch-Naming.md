---
status: accepted
date: 2025-11-19
---

# Name Branches Following a Common Standard

## Context and Problem Statement

In our development using git as a versioned source control system, we're implementing our features, improvements, bugfixes, etc. in parallel branches before they get merged into the main branch. The names of these branches can lead to a misinterpretation of their intention. A naming scheme could help.

## Considered Solutions

### Free Branch Naming

Branches can be named without any classification. The naming does not need any intention while a helpful name surely is allowed. Examples are:

- `comment-adding`
- `readme`
- `evaluate-artifact-workflow`

#### Pros

- Quite simple, fast, and free

#### Cons

- Not always helpful (see `readme`)
- Verbs describing the branch in different positions
- No grouping possible

### Detailed Prefixed Branch Names

Prefixes in fully written standard terms separated by a slash follow a well-known standard. These prefixes describe why a branch exists. Behind the slash, a small detail is used. Well-known prefixes as examples are:

- `feature/short-description`
- `bugfix/issue-42`
- `hotfix/memory-overflow`
- `improvement/branch-naming`
- `experiment/iter-usage-in-looping`
- `release/2.1.0`

#### Pros

- Intention of a branch is clearer
- Grouping makes it easier to recognize parallel work
- Prefixes help to identify urgent tasks as opposed to regular tasks (`hotfix` opposite `experiment`)

#### Cons

- Long branch names
- Intention has to be clear when branch is created

### Short Prefixed Branch Names

Similar to the detailed ones, but using abbreviations. Those could be `feat/`, `fix/`, `hot/`, `imp/`, `exp/`, or `rel/`.

#### Pros

- Shorter names without losing the benefits of the detailed names

#### Cons

- Same attention must be paid to the grouping

## Decision Outcome

A comparison between versions two and three shows how much the use of prefixes helps in classifying branches. Although proposal 3 is more compact, proposal 2 is easier to recognise and is likely to lead to fewer errors in use.

Initially, use of this standard will not be enforced technically, but will be treated as a convention. Later, local hooks will be offered that prevent developers from committing without a correctly named branch. We can offer this hook in a documented form, but it can be bypassed. Therefore, a GitHub Action can be used to prevent the merging of an incorrectly named branch.

The branch prefixes and their meanings are:

- `hotfix/` for hotfixe branches with very high priority
- `bugfix/` for regular bug fixe branches
- `feature/` for branches the which introduce new features
- `improvement/` for branches doing refactorings and improvements to the code without introducing new features or API changes
- `documentation/` for branches containing changes in the documentation and not in the code
- `release/` for the final branches to be released
- `evalaluation/` for the evaluation of new approaches and technologies; those typically will be dropped later and the results move into own issues and branches

For all designations, the issue must be mentioned, for example `feat/176-etcd-garbage-cleanup`. Additional verbs like `176-add-etcd-garbage-cleanup` are not needed. For a release, however, the identifier for example is `rel/2.0.7`.
