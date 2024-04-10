---
title: "Plugin catalog"
linkTitle: "Plugin catalog"
description: >
  Explore the catalog of Greenhouse plugins 
---

## Before you begin

This guides describes how to explore the catalog of Greenhouse _PluginDefinitions_.

While all members of an organization can see the plugin catalog, enabling, disabling and configuration _PluginDefinitions_ for an organization requires **organization admin privileges**.

## Exploring the _PluginDefinition_ catalog

The _PluginDefinition_ resource describes the backend and frontend components as well as mandatory configuration options of a Greenhouse extension.  
While the plugindefinition catalog is managed by the Greenhouse administrators and the respective domain experts, administrators of an organization can configure and tailor plugins to their specific requirements.

```text
NOTE: The UI is currently in development. For now this guides describes workflow via command line.
```

1. Run the following command to see all available _PluginDefinitions_.

   ```bash
   $ kubectl get plugindefinition
   
   NAME                      VERSION   DESCRIPTION                                                                                                  AGE
   cert-manager              1.1.0     Automated certificate management in Kubernetes                                                               182d
   digicert-issuer           1.2.0     Extensions to the cert-manager for DigiCert support                                                          182d
   disco                     1.0.0     Automated DNS management using the Designate Ingress CNAME operator (DISCO)                                  179d
   doop                      1.0.0     Holistic overview on Gatekeeper policies and violations                                                      177d
   external-dns              1.0.0     The kubernetes-sigs/external-dns plugin.                                                                     186d
   heureka                   1.0.0     Plugin for Heureka, the patch management system.                                                             177d
   ingress-nginx             1.1.0     Ingress NGINX controller                                                                                     187d
   kube-monitoring           1.0.1     Kubernetes native deployment and management of Prometheus, Alertmanager and related monitoring components.   51d
   prometheus-alertmanager   1.0.0     Prometheus alertmanager                                                                                      60d
   supernova                 1.0.0     Supernova, the holistic alert management UI                                                                  187d
   teams2slack               1.1.0     Manage Slack handles and channels based on Greenhouse teams and their members                                115d
   ```
