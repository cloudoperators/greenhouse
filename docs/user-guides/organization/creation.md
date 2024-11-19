---
title: "Creating an organization"
description: >
   Creating an organization in Greenhouse
---

## Before you begin

This guides describes how to create an organization in Greenhouse. 

During phase 1 and 2 of the roadmap Greenhouse is only open to selected early adopters.  
Please reach out to the Greenhouse team to register and create your organization via [**Slack**](https://convergedcloud.slack.com/archives/C04Q0QM40KF) or [**DL Greenhouse**](https://profiles.wdf.sap.corp/groups/651c1087132d08d3d8fac2e5/users).

## Creating an organization

An organization within the Greenhouse cloud operations platform is a separate unit with its own configuration, teams, and resources tailored to their requirements.  
These organizations can represent different teams, departments, or projects within an enterprise, and they operate independently within the Greenhouse platform.
They allow for the isolation and management of resources and configurations specific to their needs.

While the Greenhouse is build on the idea of a self-service API and automation driven platform, the workflow to onboard an organization to Greenhouse
currently involves reaching out to the Greenhouse administrators until the official go-live.  
This ensures all pre-requisites are met, the organization is configured correctly and the administrators understand the platform capabilities.

| :exclamation: Please note that the name of an organization is immutable. |
|--------------------------------------------------------------------------|

### Steps

1. **CAM Profile**  
   A CAM profile is required to configure the administrators of the organization.  
   Please include the name of the profile in the message to the Greenhouse team when signing up.


2. **SAP ID service**  
   The authentication for the users belonging to your organization is based on the OpenID Connect (OIDC) standard.  
   For SAP, we recommend using a SAP ID service (IDS) tenant.  
   Please include the parameters for your tenant in the message to the Greenhouse team when signing up.

   If you don't have a SAP ID Service tenant yet, please refer to the [SAP ID Service](/docs/user-guides/organization/sap-id) section for more information.


3. **Greenhouse organization**  
   A Greenhouse administrator applies the following configuration to the central Greenhouse cluster.  
   Bear in mind that the name of the organization is immutable and will be part of all URLs.

   ```yaml
   apiVersion: v1
   kind: Namespace
   metadata:
     name: my-organization
   ---
   apiVersion: v1
   kind: Secret
   metadata:
     name: oidc-config
     namespace: my-organization
   type: Opaque
   data:
     clientID: ...
     clientSecret: ...
   ---
   apiVersion: greenhouse.sap/v1alpha1
   kind: Organization
   metadata:
     name: my-organization
   spec:
     authentication:
       oidc:
         clientIDReference:
           key: clientID
           name: oidc-config
         clientSecretReference:
           key: clientSecret
           name: oidc-config
         issuer: https://...
       scim:
         baseURL: URL to the SCIM server.
         basicAuthUser:
           secret:
             name: Name of the secret in the same namespace.
             key: Key in the secret holding the user value.
         basicAuthPw:
           secret:
             name: Name of the secret in the same namespace.
             key: Key in the secret holding the password value.
     description: My new organization
     displayName: Short name of the organization
     mappedOrgAdminIdPGroup: Name of the group in the IDP that should be mapped to the organization admin role.
   ```

The field `mappedOrgAdminIdPGroup` is mandatory.

## Setting up Team Membership synchronization with Greenhouse
   Team Membership synchronization with Greenhouse requires access to SCIM API.

   For the Team Memberships to be created Organization needs to be configured with URL and credentials of the SCIM API. SCIM API is used to get members for teams in the organization based on the IDP groups set for teams.

   IDP group for the organization admin team should be set to the `mappedOrgAdminIdPGroup` field in the Organization configuration. It is required for the synchronization to work. IDP groups for remaining teams in the organization should be set in their respective configurations.
