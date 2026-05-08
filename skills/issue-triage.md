# Skill: Issue Triage

## Description

This skill teaches an agent how to triage GitHub issues in the `cloudoperators/greenhouse` repository. Use it whenever you are asked to triage, label, close, or comment on an issue.

The full human-readable process is in [CONTRIBUTING.md](../CONTRIBUTING.md#issue-lifecycle).

---

## Trigger

Activate this skill when asked to:

- Triage an issue
- Review a `needs-triage` issue
- Apply or suggest labels on an issue
- Determine whether an issue is ready for the backlog

---

## Step-by-Step Triage Process

### 1. Check the current state

Before acting, check:

- Does the issue have `needs-triage`? If not, it may already be triaged — confirm before making changes.
- Does the issue have enough information to make a routing decision?

### 2. Apply exactly one outcome

Remove `needs-triage` and apply one of the following — never more than one:

| Situation | Action |
|---|---|
| Issue is clear, scoped, and has acceptance criteria | Add label `backlog` |
| Issue scope is unclear or acceptance criteria are missing | Add label `needs-refinement` |
| Issue is missing details needed to evaluate | Add label `needs-more-info` + post a comment specifying exactly what is needed |
| Issue is a duplicate | Close with a comment: "Duplicate of #<number>." |
| Issue is out of scope / won't fix | Close with a short explanation comment |

**Rules:**

- Never apply `backlog` and `needs-refinement` simultaneously.
- Always post a comment when closing an issue.
- When adding `needs-more-info`, be specific — do not just say "more info needed".

### 3. Before applying `backlog` — check the Definition of Ready

Only apply `backlog` when ALL of the following are true:

- [ ] Has a clear, single-sentence problem statement
- [ ] Has testable acceptance criteria (e.g. `- [ ] criterion`)
- [ ] Has a size label applied: `size/S`, `size/M`, `size/L`, or `size/XL`
- [ ] Dependencies are identified (linked issues, or explicitly noted as none)

If any item is missing, apply `needs-refinement` instead and note what is missing in a comment.

### 4. Size estimation guide

| Label | Effort |
|---|---|
| `size/S` | Less than 1 day |
| `size/M` | 1–3 days |
| `size/L` | 3–5 days |
| `size/XL` | More than 5 days — suggest splitting the issue |

### 5. Do not touch the project board

When `backlog` is applied, a GitHub Project UI automation automatically adds the issue to the **Greenhouse Core Roadmap**. Do not manually add issues to the project.

---

## Label Reference

| Label | Applied by | Meaning |
|---|---|---|
| `needs-triage` | Automation (on open) | New issue, not yet reviewed |
| `needs-refinement` | Maintainer / agent | Needs scoping before entering backlog |
| `needs-more-info` | Maintainer / agent | Waiting on reporter for details |
| `backlog` | Maintainer / agent | Ready for sprint planning; triggers Roadmap automation |
| `bug` | Issue template | Regression or unintended behavior |
| `feature` | Issue template | New capability request |
| `size/S` | Maintainer / agent | < 1 day |
| `size/M` | Maintainer / agent | 1–3 days |
| `size/L` | Maintainer / agent | 3–5 days |
| `size/XL` | Maintainer / agent | > 5 days; suggest splitting |

---

## Example Triage Comments

**Sending to refinement:**
> Routing to refinement. The problem statement is clear, but the acceptance criteria are missing. Could you add a list of testable criteria so we can properly scope this before it enters the backlog?

**Requesting more information:**
> Marking as `needs-more-info`. To evaluate this issue we need:
>
> - The Greenhouse version where this behaviour was observed
> - The full error message or relevant log output
>
> Please update the issue and we will re-triage.

**Closing as duplicate:**
> Closing as duplicate of #42. Please follow that issue for updates. If you believe this is a distinct problem, reopen with additional context.

**Closing as out of scope:**
> Thank you for the report. After review, this falls outside the current scope of the Greenhouse core platform. We are closing this for now. If the situation changes, feel free to reopen.

**Approving to backlog:**
> Triaged. This is well-scoped with clear acceptance criteria. Moving to the backlog for sprint planning.
