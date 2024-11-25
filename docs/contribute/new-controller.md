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

> This project was generated with Kubebuilder v3, which requires Kubebuilder CLI <= v3.15.1
> Since this project does not follow the Kubebuilder v3 scaffolding structure, it is necessary to create a symlink to the main.go

```shell
ln -s ./cmd/manager/main.go main.go
```

To create a new controller, run the following command:

```shell
kubebuilder create api --group greenhouse --version v1alpha1 --kind MyResource
```

Now that the files have been generated, they need to be copied to the correct location:

```shell
mv ./apis/greenhouse/v1alpha1/myresource_types.go ./pkg/apis/v1alpha1/

mv ./controllers/greenhouse/mynewkind_controller.go ./pkg/controllers/<kind>/mynewkind_controller.go
```

After having moved the files, you need to fix the imports in the `mynewkind_controller.go` file.
Also ensure that the entry for the resource in the `PROJECT` file points to the correct location.
The new Kind should be added to the list under `charts/manager/crds/kustomization.yaml`
The new Controller needs to be registered in the controllers manager `cmd/greenhouse/main.go`.
All other generated files can be deleted.

Now you can generate all manifests with `make generate-manifests` and start implementing your controller logic.

## Implementing the Controller

Within Greenhouse the controllers implement the `lifecycle.Reconciler` interface. This allows for consistency between the controllers and ensures finalizers, status updates and other common controller logic is implemented in a consistent way. For examples on how this is used please refer to the existing controllers.

## Testing the Controller

Unit/Integration tests for the controllers use Kubebuilder's envtest environment and are implemented using Ginkgo and Gomega. For examples on how to write tests please refer to the existing tests. There are also some helper functions in the `pkg/test` package that can be used to simplify the testing of controllers.

For e2e tests, please refer to the `test/e2e/README.md`.
