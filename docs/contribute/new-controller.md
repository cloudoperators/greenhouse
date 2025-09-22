---
title: "Greenhouse Controller Development"
linkTitle: "Greenhouse Controller Development"
landingSectionIndex: false
weight: 3
description: >
  How to contribute a new controller to the Greenhouse project.
---

## Bootstrap a new Controller

> Before getting started please make sure you have read the [contribution guidelines](https://github.com/cloudoperators/greenhouse/blob/main/CONTRIBUTING.md).

Greenhouse is build using Kubebuilder as the framework for Kubernetes controllers. To create a new controller, you can use the `kubebuilder` CLI tool.

> This project was generated with Kubebuilder v4.
> It's necessary to create a symlink from `cmd/greenhouse/main.go` to `cmd/main.go` in to run the Kubebuider scaffolding commands.

```shell
ln $(pwd)/cmd/greenhouse/main.go $(pwd)/cmd/main.go
```

To create a new controller, run the following command:

```shell
kubebuilder create api --group greenhouse --version v1alpha1 --kind MyResource
```

Now that the files have been generated, they need to be copied to the correct location. The generated files are located in `api/greenhouse/v1alpha1` and `controller/greenhouse`. The correct locations for the files are `api/v1alpha1` and `pkg/controller/<kind>` respectively.
After moving the files, any imports need to be updated to point to the new locations.
Also ensure that the entry for the resource in the `PROJECT` file points to the correct location.
The new Kind should be added to the list under `charts/manager/crds/kustomization.yaml`
The new Controller needs to be registered in the controllers manager `cmd/greenhouse/main.go`.
All other generated files can be deleted.

Now you can generate all manifests with `make manifests` and start implementing your controller logic.

## Implementing the Controller

Within Greenhouse the controllers implement the `lifecycle.Reconciler` interface. This allows for consistency between the controllers and ensures finalizers, status updates and other common controller logic is implemented in a consistent way. For examples on how this is used please refer to the existing controllers.

## Testing the Controller

Unit/Integration tests for the controllers use Kubebuilder's envtest environment and are implemented using Ginkgo and Gomega. For examples on how to write tests please refer to the existing tests. There are also some helper functions in the `internal/test` package that can be used to simplify the testing of controllers.

For e2e tests, please refer to the `test/e2e/README.md`.
