# Releasing Greenhouse

This page describes the release cadence and process for the Greenhouse project.

We are following [Semantic Versioning](https://semver.org) for versioning.

> [!WARNING]
> Semantic Versioning [states](https://semver.org/spec/v2.0.0.html#spec-item-4) major version zero (0.y.z) should not be considered stable and may change at any time.

## Release Cadence

We intend for regular releases every four weeks. The release should happen in the week after the end of each sprint. For the current sprint schedule please have a look at the [roadmap project](https://github.com/orgs/cloudoperators/projects/9).

Bug fixes may be released at any time, but we will try to bundle them into the next release.

In general no feature should block a release and the `main` branch should always be stable.

Each minor release will be overseen by a release shepherd.
The responsibility of the release shepherd is to perform the release and to communicate the release to the community.

## Release Process

This is the process for releasing a new minor version of Greenhouse:

```mermaid

flowchart TD
    main[main] --"create release branch"--> releaseBranch["release/v&lt;major&gt;.&lt;minor&gt;"]
    main --"action cherry-picks bugfix"--> releaseBranch
    releaseBranch --"bump chart version & greenhouse image tags"-->releaseRCTag
    releaseBranch --"tag branch as release candidate"--> releaseRCTag["v&lt;major&gt;.&lt;minor&gt;.0-rc.*"]
    releaseRCTag --"action creates draft release"--> draftRCRelease["Draft Release Candidate"] 
    releaseRCTag --> hasBug@{ shape: diamond, label: "Bug within 3 days?" }
    hasBug -- Yes --> bugfix[Bugfix]
    bugfix --> main
    hasBug -- No --> releaseTag["v&lt;major&gt;.&lt;minor&gt;.0"]
    releaseTag --"action creates draft release"--> draftRelease["Draft Release"]
    draftRelease --> publishRelease["Publish Release"]
```

### How to tag a new release version

At the end of the sprint, the release shepherd should create a new release branch from `main`. The release branch should be named `release/v<major>.<minor>`.

1. Create a new release branch from `main` (`git checkout release-v<MAJOR.MINOR>`)
2. Push the release branch to the repository
3. Bump the version of the Greenhouse Helm chart and ensure that the greenhouse image tags are updated to the latest tag.
4. Tag the release branch with the release candidate version tag `v<major>.<minor>.0-rc.0` (`git tag v<major>.<minor>.0-rc.0`)

### How to release a new version

After pushing the release tag, there a GitHub Action is triggered to create a new draft release for the given tag. This draft release will contain the changelog and the release assets.

The release shepherd should

- review the changelog (note any breaking changes or highlight new features)
- check uploaded release assets (`greenhousectl` binaries)
- ensure the helm-charts, docker images with the release tag are uploaded to GitHub Container Registry
- add a link to the changelog since the last Greenhouse Dashboard release
- ensure the release is marked as pre-release for `rc` releases.

Once everything is in order, the release shepherd can publish the release.

In case there are bugs found for a release candidate see the [Bugfixes](#bugfixes) section on how to get fixes into the release branch.

Once the release candidate is stable, the release shepherd can create a new release tag `v<major>.<minor>.0` on the release branch and push it to the repository. A GitHub Action will run and create a new draft release for the given tag. The release shepherd should review the changelog and note any breaking changes or highlight new features. After the release notes are reviewed the release shepherd can publish the release.

### Bugfixes

Any bugs found (either during the release candidate period or after) need to be fixed on the main branch and cherry-picked to the release branch.

#### Cherry-picking with Commands

The recommended way to cherry-pick a PR to a release branch is by adding a label following the naming pattern `backport release/v0.5` to the merged PR. For example, label `backport release/v0.5` will create a cherry-pick PR targeting the `release/v0.5` branch (note that only the major.minor version is needed for the target branch).

When you add this command, a GitHub Action will automatically:

1. Create a new branch from the target release branch
2. Cherry-pick the PR's commit(s) to the new branch
3. Create a new PR with the title format `backport(release/v<major>.<minor>): Original PR Title`
4. Add appropriate labels and assign the original author

Requirements:

- The PR must be merged before using the cherry-pick command
- The target release branch must exist (e.g., `release/v0.5`)
- A member of the `@cloudoperators/greenhouse-backend` team must approve the backport

If the cherry-pick results in conflicts, the PR will be created as a draft with conflict markers included in the code, and you'll need to resolve them manually.

You can add multiple cherry-pick commands in separate comments to cherry-pick to multiple release branches.
