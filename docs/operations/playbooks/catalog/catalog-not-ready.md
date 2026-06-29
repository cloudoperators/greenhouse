---
title: "CatalogNotReady"
linkTitle: "CatalogNotReady"
landingSectionIndex: false
weight: 1
description: >
  Playbook for the GreenhouseCatalogNotReady Alert
---

## Alert Description

This alert fires when a Greenhouse catalog has not been ready for more than 15 minutes.

## What does this alert mean?

A Catalog in Greenhouse holds the available PluginDefinitions and acts as the source of truth for which plugins can be deployed within an organization. When a catalog is not ready, it indicates that the catalog source cannot be synchronized or that the catalog controller is unable to process it correctly.

This could be due to:

- The catalog source (e.g. Helm repository, OCI registry) being unavailable or unreachable
- Invalid or expired credentials for accessing the catalog source
- A misconfigured catalog resource (e.g. wrong URL, invalid reference)
- The catalog controller encountering errors during reconciliation
- Network connectivity issues between the Greenhouse operator and the catalog source

## Diagnosis

### Get the Catalog Resource

Retrieve the catalog resource to view its current status:

```bash
kubectl get catalog <catalog-name> -n <namespace> -o yaml
```

Or use kubectl describe for a more readable output:

```bash
kubectl describe catalog <catalog-name> -n <namespace>
```

### Check the Status Conditions

Look at the `status.statusConditions` section in the catalog resource. Pay special attention to:

- **Ready**: The main indicator of catalog health
- **SourceSynced**: Indicates whether the catalog source was successfully synchronized

### Check Controller Logs

Review the Greenhouse controller logs for more detailed error messages:

```bash
kubectl logs -n greenhouse -l app=greenhouse --tail=100 | grep -i catalog # requires permissions on the greenhouse namespace
```

Or access your logs sink for Greenhouse logs.

### List All Catalogs

Check if multiple catalogs are affected:

```bash
kubectl get catalog -A
```

## Additional Resources

- [Greenhouse Catalog Documentation](../../../../reference/components/catalog)
