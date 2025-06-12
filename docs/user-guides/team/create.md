---
title: "Team creation"
linkTitle: "Team creation"
description: >
  Create a team within your organization
---

## Before you begin

This guides describes how to create a team in your Greenhouse organization. 

While all members of an organization can see existing teams, their management requires **organization admin privileges**. 

## Creating a team

The team resource is used to structure members of your organization and assign fine-grained access and permission levels.

Each Team must be backed by a group in the identity provider (IdP) of the Organization.
   * IdP group should be set on the `mappedIdPGroup` field in Team configuration.
   * This, along with SCIM API configured in the Organization, allows for synchronization of Team members with Greenhouse.

```
NOTE: The UI is currently in development. For now this guides describes the onboarding workflow via command line.
```

1. To onboard a new cluster provide the kubeconfig file with a static, short-lived token.  
   It should look similar to this example:
   ```
   cat <<EOF | kubectl apply -f -
      apiVersion: greenhouse.sap/v1alpha1
      kind: Team
      metadata:
      name: <name>
      spec:
         description: My new team
         mappedIdPGroup: <IdP group name>
   EOF
   ```
