---
title: "Product design"
weight: 2
---

## Introduction

### Vision

**"Greenhouse is an extendable Platform that enables Organizations to operate their Infrastructure in a compliant, efficient and transparent manner"**

We want to build a Platform that is capable to integrate all the tools that are required to operate services in a cloud environment in a compliant, effective and transparent manner. Greenhouse aims to be the single stop for Operators of cloud-native infrastructure. The primary focus of Greenhouse is to provide a unified interface for all operational aspects and tools, providing a simple shared data model describing the support organization.

As every organization is different, using different tools and has different requirements the platform is build in an extendable fashion that allows a distributed development of plugins.

While initially developed for ApeiroRA the platform is explicitly designed to be of generic use and can be consumed by multiple organizations and teams.

### Problem Statements

#### Consolidation of Toolsuite

The operation of cloud infrastructure and applications include a large amount of tasks that are supported by different tools and automations.
Due to the high complexity of cloud environments often times a conglomerate of tools is used to cover all operational aspects. Confguration and setup of the operations toolchain is a complex and time-consuming task that often times lacks automation when it comes to on- and off-boarding people and setting up new teams for new projects.

#### Visibility of Application Specific permission concepts

At SAP, we are managing identities and access centrally. The Converged Cloud is utilizing Cloud Access Manager for this task.
While it is true that we manage who has access to which access level is defined in there it starts getting complicated if you want to figure out the actual Permission Levels on individual Applications those Access Levels are mapped to.

#### Management of organizational Groups of People

You often have groups of people that are fulfilling a organizational purpose:

* Support Groups
* Working Groups
* etc.

We have currently no way to represent and manage those groups.

#### Harmonization and Standardization of Authorization Concepts

We are missing a tool that supports teams on creating access levels and profiles following a standardized naming scheme that is enforced by design and takes away the burden of coming up with names for access levels and user profiles/roles.

#### Single Point of Truth for Operations Metadata of an Organization

For automations, it is often critical to retrieve Metadata about an Organizational Unit:

* Who is member of a certain Group of people, that is maybe not reflecting the HR View of a Team?
* Which Tool is giving me access to data x,y,z?
* What action items are due and need to get alerted on?
* Does component x,z,y belong to my organization?
* etc.
Currently, we do not have a single point of Truth for this kind of metadata and have to use a vaierity of Tools.

## Terms

This section lists down Terms and description to Terms to ensure a common languague when talking in context of Greenhouse.

| Term | Description |
| -------- | -------- |
| PluginDefinition     | A Greenhouse PluginDefinition provides additional features to the platform. It consists of a [Juno microfrontend](https://github.com/cloudoperators/juno) that integrates with the Greenhouse UI AND / OR  a backend component.     |
| PluginDefinitionSpec     | YAML Specification describing the PluginDefinition. Contains reference to components (UI App, Helm Chart) that need to be installed. Describes mandatory, optional and default configuration values.     |
| Plugin  | A Plugin is a specific instance of a PluginDefinition in a Greenhouse Organization. References the PluginDefinitionSpec and actual configuration values     |
| PluginPreset | A PluginPreset is managing the configuration of Plugins for a set of Clusters. Adding a new Cluster to the set will automatically create the Plugin.     |
| Organization     | An Organization is the overarching container for all resources in Greenhouse. One Greenhouse installation can contain multiple Organization. Each Organization is isolated on the level of Kubernetes Namespace.     |
| Team     | A Team is part of an Organization and consists of users.     |
| TeamRole     | A TeamRole that can be assigned to Teams. Roles are a static set that can used by UIs to allow/disallow actions (admin, viewer, editor)     |
| TeamRoleBinding | A TeamRoleBinding assigns a TeamRole to a Team. Currently it is used to provide access to onboarded Kubernetes Clusters.
| Cluster     | A specific Kubernetes cluster to which an Organization and its members have access to and which can be onboarded to Greenhouse.     |
| ClusterKubeConfig | A ClusterKubeConfig contains the KubeConfig of a Cluster. Greenhouse will provide a central registry for Clusters consumable via CLI for easier cluster access. |
| Identity Provider     | Central authentication provider that provides authentication for the User of on Organization. Used by the UI and APIServer to authenticate users.     |
| Authorization Provider     | External system that provides authorization, e.g. team assignments for users     |
| Greenhouse APIServer     | Central Kubernetes APIServer for Greenhouse. Kubernetes APIServer with Greenhouse CRDs deployed.     |
| OIDC Group     | A Group provided by the OIDC Provider (Identity Provider) userinfo with the JWT Token.     |

## User Profiles

Every Application has Users / Stakeholders, so has Greenhouse. The User Profiles mentioned here give a abstract overview of considered Users / Stakeholders for the Application and the Goals and Tasks of them in context of the Platform.

### Greenhouse admin

Administrator of a Greenhouse installation.

#### Goals

* Ensure overall function of the Greenhouse plattform

#### Tasks

* Create Organizations
* Enable PluginDefinitions
* Operates central infrastructure (Kubernetes cluster, operator, etc.)
* Assign initial Organization admins

### Organization admin

Administrator of a Greenhouse Organization

#### Goals

* Manage organization-wide resources

#### Tasks 

* configure Plugins for the Organization
* Create and manage Teams for the Organization
* Onboard and manage Kubernetes clusters for the Organization

### Organization Cluster admin

Administrator of a Greenhouse Organization's Kubernetes cluster

#### Goals

* Manage an Organizations Clusters

#### Tasks

* configure Plugins for the Organization
* Create and manage Team access to the Organization Cluster
* Onboard and manage Kubernetes clusters for the Organization

### Organization Plugin admin

Administrator of a Greenhouse Organization's Plugins

#### Goals

* Manage an Organizations Plugins

#### Tasks

* configure Plugins for the Organization

### Organization member

Member of an Organization that accesses the UI to do ops/support tasks. Is member of one ore more teams of the organization.
By default members have view permissions on Organization resources.

#### Goals

* Provide ops/support for the services of the organization

#### Tasks 

* Highly dependend on Team membership and Plugins configured for the Organization
* Examples:
  * Check alerts for Teams the user is assigned to
  * Check policy violations for deployed resources owned by the Users Team
  * Check for known vulnerabilites in services

### Plugin developer

A plugin developer is developing PluginDefinitions for Greenhouse.

#### Goals

* Must be easy to create PluginDefinitions
* Can create and test PluginDefinitions independently
* Greenhouse provides tooling to assist creating, validating, etc. of PluginDefinitions
* Publishing the PluginDefintion to Greenhouse requires admin permissions.

#### Tasks 

* PluginDefintion Development
  * Juno UI Development
  * PluginDefinition backend development

### Auditor

An Auditor audits Greenhouse and/or Greenhouse PluginDefinitions for compliance with industry or regulatory standards.

#### Goals

* Wants to see that we have a record for all changes happening in greenhouse
* Wants to have access to resources required to audit the respective Plugin

#### Tasks 

* Performs Audits

### Greenhouse Developer

Develops parts of the Greenhouse platform (Kubernetes, Greenhouse app, Greenhouse backend, ...)

#### Goals

* Provide Greenhouse framework

#### Tasks 

* Provides framework for plugin developers to develop plugins
* Develops Greenhouse framework components (UI or backend)

## User Stories 

----------

The User Stories outlined in this Section have the target to archive a common Understanding of the capabilities/functionalities the Platform wants to archive and the functional requirements that come with those. The Integration / Development of Functionalities is not going to be strictly bound to User Stories and they are rather used as an orientation to ensure that envisioned capabilities are not getting Blocked due to implementation details. 

The details of all User Stories are subject to change based on the results of Proof of Concept implementations, User feedback or other unforseen factors.

----------

### Auditor 

----------

#### Auditor 01 - Audit Logging 

As an Auditor, I want to see who did which action and when to verify that the Vulnerability and Patch management process is followed according to company policies and that the platform is functioning as expected.

##### Acceptance Criteria

* Every state-changing action is logged into an immutable log collection, including:
    * What action was performed
    * Who performed the action
    * When was the action performed
* Every authentication to the platform is logged into an immutable log collection, including:
    * Who logged in
    * When was the login 

----------

### Greenhouse Admin 

#### Greenhouse Admin 01 - Greenhouse Management Dashboard

As an Greenhouse Admin, I want a central Greenhouse Management Dashboard that allows me to navigate through all organization-wide resources to be able to manage the Platform.

##### Acceptance Criteria

* Assuming I am on the Greenhouse Management Dashboard view, i can:
    * See all Plugins, including the enabled version
    * Order not enabled Plugins by last Update Date
    * Plugins are Ordered by the Order Attribute
    * The order attribute is a numeric value that can be changed to reflect a different ordering of the Plugin:
        * 1 is ordered before 2 etc.
        * The order attribute is used as well to order the Plugins on the Navigation Bar
    * Navigate to "Plugin Detail View" by clicking a Plugin
    * See all Organizations, including:
        * Number of Organization Admins
        * Number of Organization Members
    * Navigate to organization creation view by clicking "Create Organization"
    * Navigate to Organization Detail View by clicking a Organization
* Only Greenhouse Admin's should be able to see the Dashboard
* The Navigation item to the Greenhouse Management Dashboard should only be visible to Greenhouse admins

#### Greenhouse Admin 02 - Organization Creation View

As a Greenhouse Admin, I want a Organization Creation view that allows me to create a new Organization

##### Acceptance Criteria

* Assuming I am on the Organization creation View, i can:
    * Give a unique name for the organisation
    * Provide a short description for the organization
    * Provide a OIDC Group that gives Organization Admin Privileges

### Greenhouse Admin & Organization Admin 

#### Greenhouse Admin & Organization Admin 01 - Organization Detail View 

As a Greenhouse Admin or Organization Admin, I want an Organization Detail view that allows me to view details about an organization

##### Acceptance Criteria

* Assuming I am on the organization detail View, i can:
    * Can see the organization details (name/description)
    * See a list of teams created for this organization
    * See the list of active plugins
    * Add Plugins to the organization by clicking "add Plugin"
    * Change the Organization Admin Role Name

#### Greenhouse Admin & Organization Admin 02 - Plugin Detail View

As an Greenhouse Admin or Organizatioin Admin, I want a Plugin Detail view that allows me to see  Plugin Details to be able to see details about the plugin.

##### Acceptance Criteria

* Assuming I am on the Plugin Detail View, I can:
    * see the plugin name
    * see the plugin description
    * see the last update date
    * see the release reference
    * see the ui release refrence
    * see the helm chart reference
    * see the ordering attribute
    * see configuration values for the plugin
    * set the configuration values for the current organization
    * see a change log
    * see the actually released (deployed to greenhouse) version

### Organization Admin 

#### Organization Admin 01 - Organization Managment Dashboard

As an Organization Admin, I want to have an Dashboard showing me the most relevant information about my Organization to be able to manage it efficently. 

##### Acceptance Criteria

* Assuming I am on the organisation management dashboard
    * I can see a list of all teams in my organization
    * I can see a list of configured plugins
    * I can click a "add plugin" button to add a new plugin
    * I can see a list of registered clusters
    * I can click a "add cluster" button to register a cluster


#### Organization Admin 02 - Plugin Configuration View

As an organization admin, I want a Plugin configuration view that allows me to enable and configure greenhouse plugins for my organization

##### Acceptance Criteria

* Assuming I am on the Plugin configuration View, I can: 
    * select the type of plugin I want to configure
    * enable/disable the plugin (for my org)
    * remove the plugin (when already added)
    * manage configuration options specific for the plugin

#### Organization Admin 03 - Cluster registration View

As an organization admin, I want a Cluster registration view to onboard kubernetes clusters into my organization.

##### Acceptance Criteria

* Assuming I am on the cluster registration view, I can: 
    * Get instructions how to register a kubernetes clusters
    * give a name and description for the registered cluster
* After executing the provided instructions I get feedback that the cluster has been successfully registered
* A cluster can be registered to exactly one organization

#### Organization Admin 04 - Cluster detail View

As an organization admin, I want a Cluster detail view to get some information about a registered cluster

##### Acceptance Criteria

* Assuming I am on the cluster detail view, I can: 
    * see basic details about the cluster:
        * name
        * api url
        * version
        * node status
    * de-register the cluster from my organization


#### Organization Admin 05 - Team Detail View 

| :exclamation:  User Story details depending on final decision of ADR-01  |
|-----------------------------------------|

As an organization admin, i want to have a Team Detail View, with the option to configure teams based on role mapping to be to manage teams within my organization without managing the permission administration itself on the Platform

##### Acceptance Criteria

* Assuming I am on the Team detail view, i can:
    * Change the Name of the Team
    * change the description of the team
    * Define a single OIDC Group  that assign you this team
    * Define The Greenhouse Role that you get within Greenhouse if you are a member of the team
* On Login of a User into an Organization the Platform verifies if the User has ALL required roles


#### Organization Admin 05 - Team Creation View 

| :exclamation:  User Story details depending on final decision of ADR-01  |
|-----------------------------------------|

As an organization admin, i want to have
a Team Creation View, to be able to create a new Team

##### Acceptance Criteria

* Assuming I am on the Team Creation view, I can::
    * Set the name of the Team
    * Set a description of the Team
    * Set a OIDC Group Name that assigns users to this team


### Organization Member 

#### Organization Member 01 - Unified task inbox 

As an organization member, I want a task inbox that shows my open tasks from all enabled plugins that need my attention to be on top of my tasks to fulfill across all plugins

##### Acceptance Criteria

* Assuming I am on the task inbox:
    * I can a list of open task accross all plugins that need attention
    * clicking on an open task jumps in the plugin specific UI the task belong to
    * I can sort open tasks by name, plugin or date

### Plugin Developer 

As a Plugin Developer, I want to have a seperate Repository for my Plugin which I can own and use to configure plugin internals to have control over the Development efforts and configuration of the Plugin

#### Plugin Developer 01 - Decentrally Managed Plugin

As a Plugin Developer, I want to have a seperate Repository for my Plugin which I can own and use to configure plugin internals to have control over the Development efforts and configuration of the Plugin

##### Acceptance Criteria

* Plugin lives on his own Github Repository
* Versions are managed via Github Releases using Tags and the release to Greenhouse is managed by the Plugin:
    * The version to be pulled by Greenhouse is managed by the Plugin Developer.
* I can configure the Plugin Configuration over a greenhouse.yml in the root of the repository, which at least includes (mandatory):
    * description: ...
    * version: ...
* There are optional attributes in the greenhouse.yml:
    * icon: which if it has a valid absolute path to an image file on the repository makes the icon selectable as an icon in the plugin detail view (GA02)
    * describes available configuration options that attributes that are required for the plugin to function
    * I can specify a reference to a UI App
    * I can specify a reference to Helm Charts


#### Plugin Developer 02 - Plugin Role Config

As a Plugin Developer i want a section within the Greenhouse.yml metadata, named "Roles" where i can setup Roles used by my Plugin

##### Acceptance Criteria

| :warning:  User Story details depending on final decision of ADR-01 and are therefore not further described here  |
|-----------------------------------------|

#### Plugin Developer 03 - Spec Schema Validation

As a Plugin Developer I want to have the possibility to validate the schema of my greenhouse.yaml to be able to catch errors within my specification early.

##### Acceptance Criteria

* The schema check should support IDE's
* The schema check should be automate-able and be integrate-able to pre-commit hooks and quality gates
* A version with a broken schema should not be release on greenhouse even when pushing for a pull of the release
* It should be visible on the Plugin detail view when an invalid schema was released with a recent version

#### Plugin Developer 04 - Config Value Validation

As a Plugin Developer I want to have the possibility to write custom regex checks for configuration options of my plugin that include the check to be performed on a field and an error message to be shown if configured wrong by an organization to support organization admins on configuring my plugin

##### Acceptance Criteria

* The validation rules should be controlled by Plugin Developer
* The validation should happen on the frontend before submitting a configuration
* The error message should be shown when a config value is provided wrong

#### Plugin Developer 05 -  Plugin development tooling

As a plugin developer I want to have an easy setup for developing and testing greenhouse plugins

##### Acceptance Criteria

* Dev environment available within X Time
* Possible to have a working local setup with a "mock greenhouse apiserver"
* Has a fully working Bootstrap Project that includes Backend and Frontend which can be run locally immediately
* Has documentation

## Product Stages

### Overview

This Section gives an overview of the different early stages of the Platform that are beeing developed and which functional requirements need to get fulfilled within those stages. 

### Proof of Concept (POC)

The Proof of Concept is the stage where fundamental Framework/Platform decisions are proven to be working as intended. At this Stage the Platform is not suitable to be used by the intended audience yet but most necessary core functionalities are implemented. 

The desired functionalities in this phase are: 

* Frontend with Authentication
* Authorization within Greenhouse (Greenhouse Admin, Org Admin, Org Member)
* Team Management (without UI)
* Org Management (without UI)
* Greenhouse Admin User Stories (without UI)
* Dummy Plugin
    * with configuration spec
* Plugin Development Setup (without Documentation)
* Plugin Versioning & Provisioning (without UI)


### Minimal viable product (MVP)

This stage is considered to be the earliest stage to open the Platform for General use.  
In addition to the PoC functionalities we expect the following requirements to be fulfilled: 

* Integrated 3 Plugins: 
    * Supernova (Alerts)
    * DOOP (Violations)
    * Heureka   
      NOTE: Heureka was excluded from MVP as the Heureka API is only available at a later point in time.
* Team management
* Organization management
