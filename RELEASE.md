# Releasing Greenhouse

This page describes the release cadence and process for the Greenhouse project.

We are following [Semantic Versioning](https://semver.org) for versioning.

> [!WARNING]
> Semantic Versioning [states](https://semver.org/spec/v2.0.0.html#spec-item-4) major version zero (0.y.z) should not be considered stable and may change at any time.

## Release Cadence

We intend for regular releases every four weeks. The release is should happen in the week after the end of each sprint. For the current sprint schedule please have a look at the [roadmap project](https://github.com/orgs/cloudoperators/projects/9).

Bug fixes may be released at any time, but we will try to bundle them into the next release.

In general no feature should block a release and the `main` branch should always be stable.

Each minor release will be overseen by a release shepherd.
The responsibility of the release shepherd is to perform the release and to communicate the release to the community.

## Release Process

This is the process for releasing a new minor version of Greenhouse:

```mermaid

flowchart TD
    start[Start] --> releaseCandidate["Release
    v&lt;major&gt;.&lt;minor&gt;.0-rc.0"]
    releaseCandidate --> hasBug[Bug after 3 workdays?]
    hasBug -- Yes --> bugfix[Bugfix]
    bugfix --> newReleaseCandidate[Release 
     v&lt;major&gt;.&lt;minor&gt;.0-rc.++]
    newReleaseCandidate --> hasBug
    hasBug -- No --> e[End]
```

### How to tag a new release version

At the end of the sprint, the release shepherd should create a new release branch from `main`. The release branch should be named `release/v<major>.<minor>`.

This release branch should then be pushed to the repository.

This branch can then be tagged with the release candidate version tag `v<major>.<minor>.0-rc.0`.

// TODO: Add instructions for the changelog based on GH Action

Any bugs found (either during the release candidate period or after) need to be fixed on the main branch and cherry-picked to the release branch.

Once the release candidate is stable, the release shepherd can create a new release tag `v<major>.<minor>.0` and push it to the repository.

### How to release a new version

After pushing the release tag, there will be a GitHub Action that will run and create a new draft release for the given tag.

The release shepherd should

- review the changelog
- check uploaded release assets (helm-charts, docker images, binaries, etc. )
- ensure the release is marked as pre-release for `rc` releases.

Once everything is in order, the release shepherd can publish the release.
