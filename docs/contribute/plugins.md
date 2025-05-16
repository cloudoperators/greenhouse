---
title: "Contributing a Plugin"
linkTitle: "Contributing a Plugin"
landingSectionIndex: false
weight: 2
description: >
  Contributing a Plugin to Greenhouse
---

# What is a Plugin?

A Plugin is a key component that provides additional features, functionalities and may add new tools or integrations to the Greenhouse project.  
They are developed de-centrally by the domain experts.  
A YAML specification outlines the components that are to be installed and describes mandatory and optional, instance-specific configuration values.

It can consist of two main parts:

1. Juno micro frontend  
   This integrates with the Greenhouse dashboard, allowing users to interact with the Plugin's features seamlessly within the Greenhouse UI.

2. Backend component  
   It can include backend logic that supports the Plugin's functionality.

# Contribute

Additional ideas for plugins are very welcome!  
The Greenhouse plugin catalog is defined in the [Greenhouse extensions repository](https://github.com/cloudoperators/greenhouse-extensions).  
To get started, please file an issues and provide a concise description of the proposed plugin [here](https://github.com/cloudoperators/greenhouse-extensions/issues).

A Greenhouse plugin consists of a juno micro frontend that integrates with the Greenhouse UI and/or a backend component described via Helm chart.  
Contributing a plugin requires the technical skills to write Helm charts and proficiency in JavaScript.  
Moreover, documentation needs to be developed to help users understand the plugin capabilities as well as how to incorporate it.  
Additionally, the plugin needs to be maintained by at least one individual or a team to ensure ongoing functionality and usability within the Greenhouse ecosystem.

## Development

Developing a plugin for the Greenhouse platform involves several steps, including defining the plugin, creating the necessary components, and integrating them into Greenhouse.  
Here's a high-level overview of how to develop a plugin for Greenhouse:

1. **Define the Plugin**:

   - Clearly define the purpose and functionality of your plugin.
   - What problem does it solve, and what features will it provide?

2. **Plugin Definition (plugindefinition.yaml)**:

   - Create a `plugindefinition.yaml` ([API Reference](https://cloudoperators.github.io/greenhouse/docs/reference/api/#greenhouse.sap/v1alpha1.PluginDefinition)) file in the root of your repository to specify the plugin's metadata and configuration options. This YAML file should include details like the plugin's description, version, and any configuration values required.
   - Provide a list of `PluginOptions` which are values that are consumed to configure the actual `Plugin` instance of your `PluginDefinition`.
     Greenhouse always provides some global values that are injected into your `Plugin` upon deployment:
     - `global.greenhouse.organizationName`: The `name` of your `Organization`
     - `global.greenhouse.teamNames`: All available `Teams` in your `Organization`
     - `global.greenhouse.clusterNames`: All available `Clusters` in your `Organization`
     - `global.greenhouse.clusterName`: The `name` of the `Cluster` this `Plugin` instance is deployed to.
     - `global.greenhouse.baseDomain`: The base domain of your Greenhouse installation
     - `global.greenhouse.ownedBy`: The owner (usually a owning `Team`) of this `Plugin` instance

3. **Plugin Components**:

   - Develop the plugin's components, which may include both frontend and backend components.
   - For the frontend, you can use Juno microfrontend components to integrate with the Greenhouse UI seamlessly.
   - The backend component handles the logic and functionality of your plugin. This may involve interacting with external APIs, processing data, and more.

4. **Testing & Validation**:

   - Test your plugin thoroughly to ensure it works as intended. Verify that both the frontend and backend components function correctly.
   - Implement validation for your plugin's configuration options. This helps prevent users from providing incorrect or incompatible values.
   - Implement Helm Chart Tests for your plugin if it includes a Helm Chart. For more information on how to write Helm Chart Tests, please refer to [this guide](/greenhouse/docs/user-guides/plugin/plugin-tests).

5. **Documentation**:

   - Create comprehensive documentation for your plugin. This should include installation instructions, configuration details, and usage guidelines.

6. **Integration with Greenhouse**:

   - Integrate your plugin with the Greenhouse platform by configuring it using the Greenhouse UI. This may involve specifying which organizations can use the plugin and setting up any required permissions.

7. **Publishing**:

   - Publish your plugin to Greenhouse once it's fully tested and ready for use. This makes it available for organizations to install and configure.

8. **Support and Maintenance**:

   - Provide ongoing support for your plugin, including bug fixes and updates to accommodate changes in Greenhouse or external dependencies.

9. **Community Involvement**:
   - Consider engaging with the Greenhouse community, if applicable, by seeking feedback, addressing issues, and collaborating with other developers.
