---
title: "CatalogNotReady"
linkTitle: "CatalogNotReady"
landingSectionIndex: false
weight: 1
description: >
  Playbook for the CatalogNotReady Alert
---

## Alert Description

This alert fires when a Catalog has not been ready for more than 15 minutes.

## What does this alert mean?

A Catalog fetches PluginDefinitions from Git repositories and applies them to the Organization. For each entry in `.spec.sources` Greenhouse creates a chain of Flux resources: a GitRepository, an ArtifactGenerator, an ExternalArtifact and a Kustomization. The Catalog is ready only when every object of every source is ready.

When the Catalog is not ready, changes from the failing source may not reach the Organization. A failure in one source does not block the other sources, and already applied PluginDefinitions stay as they are.

The `reason` label on the alert gives a first hint:

- `SecretMissing`: the Secret referenced in `.spec.sources[].secretName` does not exist in the Organization namespace
- `AuthError`: the GitRepository cannot authenticate against the Git host, the credentials are invalid or expired
- `Unknown`: any other failure, check the Catalog status as described below

## Diagnosis

### Get the Catalog Resource

```bash
kubectl get catalog <catalog-name> -n <organization-namespace>
kubectl describe catalog <catalog-name> -n <organization-namespace>
```

### Check the Status Conditions

Look at the `Ready` condition in `.status.statusConditions`. Its reason tells where the reconciliation stopped:

- `CatalogEmptySources`: `.spec.sources` is empty
- `CatalogSourceValidationFail`: a source repository URL is not of the form `https://<host>/<owner>/<repo>`, or the same repository and ref are listed twice
- `OrphanedObjectCleanUpFail`: Flux objects of a removed source could not be deleted
- `CatalogNotReady`: at least one Flux object of a source is not ready, or Greenhouse could not create or update it. A missing Secret or an option override that names an option the PluginDefinition does not have is reported only in this message, the Flux object keeps its previous state and can still be ready

### Check the Inventory

`.status.inventory` lists the Flux objects of each source with their ready state and message. Find the first object that is not ready, in this order:

- **GitRepository**: the repository could not be cloned. Common causes are a wrong URL, a branch, tag or SHA that does not exist, or invalid credentials in the referenced Secret.
- **ArtifactGenerator**: the files in `.spec.sources[].resources` could not be collected, for example a path that does not exist in the repository.
- **ExternalArtifact**: the artifact was not generated yet, this follows from a failing ArtifactGenerator.
- **Kustomization**: the PluginDefinitions could not be applied. Common causes are an invalid PluginDefinition manifest or missing permissions.

Inspect the failing object for its full status and events:

```bash
kubectl describe gitrepository <name> -n <organization-namespace>
kubectl describe artifactgenerator <name> -n <organization-namespace>
kubectl describe kustomization <name> -n <organization-namespace>
```

### Trigger a Reconciliation

After fixing the cause, for example updating the Secret or pushing a fix to the repository, trigger an immediate reconciliation instead of waiting for the next interval:

```bash
kubectl annotate catalog <catalog-name> -n <organization-namespace> greenhouse.sap/reconcile="$(date +%s)" --overwrite
```

Also check that the Catalog is not suspended by the `greenhouse.sap/suspend` annotation.

## Additional Resources

- [Catalog Documentation and Troubleshooting](../../../../reference/components/catalog)
- [Flux GitRepository Documentation](https://fluxcd.io/flux/components/source/gitrepositories/)
- [Flux Kustomization Documentation](https://fluxcd.io/flux/components/kustomize/kustomizations/)
