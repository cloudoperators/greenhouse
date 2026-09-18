    # Renovate Merge

Review open Renovate PRs in this repository, summarize the changes, ask for approval, wait for all CI checks to pass, then merge the approved PRs.

## Steps

1. **List open Renovate PRs**

Run the following to list PRs ordered by size label from smallest to largest (XS → S → M → L → XL):
```bash
gh pr list --state open --author "renovate[bot]" --repo cloudoperators/greenhouse --json number,title,labels,headRefName,createdAt | \
  jq 'def size_order: . as $l | ["XS","S","M","L","XL"] | index($l) // 99;
  sort_by(.labels | map(.name | select(startswith("size/"))) | first // "" | ltrimstr("size/") | size_order) |
  .[] | [(.number | tostring), .title, (.labels | map(.name) | join(", ")), .headRefName] | @tsv'
```

If there are no open Renovate PRs, report that and stop.

2. **Review and summarize each PR**

For each open PR, fetch its details:
```
gh pr view <PR_NUMBER> --repo cloudoperators/greenhouse
```

Also fetch the diff to understand what changed:
```
gh pr diff <PR_NUMBER> --repo cloudoperators/greenhouse
```

Group the PRs by category (e.g. Go modules, GitHub Actions, tooling/lint, Kubernetes packages, Flux packages, major upgrades) and present a concise summary table with:
- PR number and title
- What is being updated (dependency name, old version → new version)
- Category (patch / minor / major)
- Any notable risks (e.g. major version bumps, breaking changes)

3. **Ask which PRs to process**

Ask the user which PRs they want to merge. Present the list and wait for the user to confirm specific PR numbers or say "all".

4. **For each selected PR: show diff, ask for confirmation, update if needed, wait for checks, then merge**

Process each PR one at a time in the following order:

**a) Show the diff and ask for confirmation**

Fetch and display the diff for the PR:
```bash
gh pr diff <PR_NUMBER> --repo cloudoperators/greenhouse
```

Present a concise summary of what changed (files modified, version bumps) and explicitly ask the user: "Do you want to approve and merge PR #<PR_NUMBER>?" Wait for the user to respond before proceeding.

- If the user says **"approve"** (or yes/merge): approve the PR and enable auto-merge (see steps b and c below).
- If the user says **"skip"** (or no/decline): skip to the next PR.

**b) Check if the PR needs updating (rebasing)**

```bash
gh pr view <PR_NUMBER> --repo cloudoperators/greenhouse --json mergeStateStatus,mergeable
```

If `mergeStateStatus` is `BEHIND` or `mergeable` is `CONFLICTING`, rebase the PR against the base branch by posting a rebase comment:
```bash
gh pr comment <PR_NUMBER> --repo cloudoperators/greenhouse --body "@renovate rebase"
```
Then wait 30 seconds and poll until the PR is no longer behind before continuing.

**c) Approve the PR and enable auto-merge**

First, approve the PR:
```bash
gh pr review <PR_NUMBER> --repo cloudoperators/greenhouse --approve
```

Then poll CI checks every 60 seconds until all checks reach a terminal state before merging. Only `pass` and `skipping` are considered success; `fail`, `cancelled`, and `timed_out` are failures; `queued`, `pending`, and `in_progress` keep the loop waiting:

```bash
while true; do
  STATUS=$(gh pr checks <PR_NUMBER> --repo cloudoperators/greenhouse 2>&1)
  NOT_TERMINAL=$(echo "$STATUS" | grep -cE "\t(queued|pending|in_progress)\t" || true)
  FAILING=$(echo "$STATUS" | grep -cE "\t(fail|cancelled|timed_out)\t" || true)
  PASSING=$(echo "$STATUS" | grep -cE "\t(pass|skipping)\t" || true)
  TOTAL=$(echo "$STATUS" | grep -cE "^\S" || true)
  echo "[$(date '+%H:%M:%S')] PR #<PR_NUMBER> — Total: $TOTAL, Passing: $PASSING, Waiting: $NOT_TERMINAL, Failing: $FAILING"
  if [ "$NOT_TERMINAL" -eq 0 ] && [ "$FAILING" -eq 0 ] && [ "$TOTAL" -gt 0 ]; then
    echo "All checks passed — proceeding to merge PR #<PR_NUMBER>."
    break
  elif [ "$NOT_TERMINAL" -eq 0 ] && [ "$FAILING" -gt 0 ]; then
    echo "Some checks failed on PR #<PR_NUMBER>. Skipping merge."
    echo "$STATUS" | grep -E "\t(fail|cancelled|timed_out)\t"
    exit 1
  fi
  sleep 60
done
```

If any check failed, cancelled, or timed out, skip this PR and report the failures.

Once all checks pass, merge the PR:
```bash
gh pr merge <PR_NUMBER> --repo cloudoperators/greenhouse --squash
```

Then verify it was actually merged:
```bash
gh pr view <PR_NUMBER> --repo cloudoperators/greenhouse --json state,mergedAt \
  | jq 'if .state == "MERGED" then "PR #<PR_NUMBER> merged at \(.mergedAt)" else "PR #<PR_NUMBER> not merged — state: \(.state)" end'
```

Report the actual merge state from this output.

5. **Final report**

After processing all approved PRs, summarize:
- Which PRs were successfully merged
- Which PRs were skipped due to failing checks
- Any PRs that were not approved by the user
 