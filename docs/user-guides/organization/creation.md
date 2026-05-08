---
title: "Creating an organization"
description: >
   Creating an organization in Greenhouse
---

## Before you begin

This guides describes how to create an organization in Greenhouse.

## Creating an organization

An organization within the Greenhouse cloud operations platform is a separate unit with its own configuration, teams, and resources tailored to their requirements. 
These organizations can represent different teams, departments, or projects within an enterprise, and they operate independently within the Greenhouse platform.
They allow for the isolation and management of resources and configurations specific to their needs. Since each Organization will receive its own Kubernetes Namespace within the Greenhouse cluster.

While Greenhouse is build on the idea of a self-service API and automation driven platform, the workflow to onboard an organization to Greenhouse involves reaching out to the Greenhouse administrators.
This ensures all pre-requisites are met, the organization is configured correctly and the administrators of the Organization understand the platform capabilities.

| :exclamation: Please note that the name of an organization is immutable. |
|--------------------------------------------------------------------------|

### Steps

1. **IdP Group**
   An IdP Group is required to configure the administrators of the organization. Onboarding without an Group is not possible, as this would leave the organization without any administrators having access.
   Please include the name of the IdP Group in the message to the Greenhouse team when signing up.

2. **Identity Provider**  
   The authentication for the users belonging to your organization is based on the OpenID Connect (OIDC) standard.  
   Please include the parameters for your OIDC provider in the message to the Greenhouse team when signing up.

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

## Setting up Team members synchronization with Greenhouse

   Team members synchronization with Greenhouse requires access to SCIM API.

   For the members to be reflected in a Team's status, the created Organization needs to be configured with URL and credentials of the SCIM API. SCIM API is used to get members for teams in the organization based on the IDP groups set for teams.

   IDP group for the organization admin team must be set to the `mappedOrgAdminIdPGroup` field in the Organization configuration. It is required for the synchronization to work. IDP groups for remaining teams in the organization should be set in their respective configurations - Team's `mappedIdpGroup` field.

## Next Steps

- [Organization reference](./../../../reference/components/organization)
