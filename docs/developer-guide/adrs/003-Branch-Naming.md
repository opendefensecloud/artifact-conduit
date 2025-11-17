---
status: in decission
date: 2025-11-14
---

# Name Branches Following a Common Standard

## Context and Problem Statement

In our developing using git as a versioned source control system we're implementing our features, improvements, bugfixes etc. in parallel branches before they'll get merged into the main branch. The names of these branches can lead to a misinerpretation of it's intention. A naming status could help.

## Considered Solutions

### Free Branch Naming

Branches can be named without any classification. The naming does not need any intention while a helpful name surely is allowed. Examles are

- `comment-adding`
- `readme`
- `evaluate-artiface-workflow`

#### Pros

- Quite simple, fast, and free

#### Cons

- Not allways helpfule (see `readme`)
- Verbs describing the branch on different positions
- No grouping possible

### Detailed Prefixed Branch Names

Prefixes in full writtend standard terms seperated by a slash follow a well known standard. These prefixis describe, why a branch exists. Behind the slash a small detail is used. Well known prefixes as examples are

- `feature/short-description`
- `bugfix/issue-42`
- `hotfix/memory-overflow`
- `improvement/branch-naming`
- `experiment/iter-usage-in-looping`
- `release/2.1.0`

#### Pros

- Intention of a branch is more clear
- Grouping makes it easier to recognise parallel work
- Prefixes help to identify urgent tasks as opposed to regular tasks (`hotfix` opposite `experiment`)

#### Cons

- Long branch names
- Intention has to be clear when branch is created

### Short Prefixed Branch Names

Similar to the detailed ones, but using abbreviations. Those could be `feat/`, `fix/`, `hot/`, `imp/`, `exp/`, or `rel/`.

#### Pros

- Shorter names w/o loosing the benefits of the detailed names

#### Cons

- Same attention must be paid to the grouping

## Decission Outcome

As a compromise, proposal 3 was initially introduced without compulsion. Its use is based on the conventions in the still manageable team and the limited number of developers who can perform reviews. If the prefixes are not used, an initial note is sufficient.

In a second step, local hooks could be made available and documented for the project's supporters, and a GitHub Action could be defined that prohibits the merging of such branches. These steps are optional for the future.

The branch prefixes and their meanings are:

- `hot/` for hotfixes with very high priority,
- `bug/` for regular bug fixes,
- `feat/` for the introduction of new features,
- `imp/` for refactorings and improvements to the code without introducing new features or API changes,
- `eval/` for the evaluation of new approaches and technologies,
- `doc/` for changes in the documentation and not in the code, and
- `rel/` for releases to be released.

For all designations, the issue must be mentioned, for example `feat/176-etcd-garbage-cleanup`. Additional verbs like `176-add-etcd-garbage-cleanup` are not needed. For a release, however, the identifier for example it is `rel/2.0.7`.
