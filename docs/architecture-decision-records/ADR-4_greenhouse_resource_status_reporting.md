# ADR-4 Greenhouse Resource States

## Decision Contributors

- Ivo Gosemann
- Uwe Mayer

## Status

- Proposed

## Context and Problem Statement

Greenhouse contains controllers for several custom resources such as `Cluster`, `Plugin`, `Team`, `TeamMembership`, etc.

These objects need a unified approach for reporting and maintaining their states.

Therefore this ADR adresses two concerns:

1. provide a single source of truth for maintaining resource states
2. provide a common guideline with best practices on reporting states during reconciliation

## Decision Drivers

- Uniformity:

  - All resource states should be accessible the same way.

- Expandability:

  - New resources with respective requirements should be easily integrated into the existing structure

- Ease of use:
  - Interaction with the provided structure should be clear and as easy as possible

## Decision

We will follow [kubernetes SIG architecture advice](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties) and many other renowned projects introducing a greenhouse condition.

For more diffentiating states than `Ready=true/false/unkown` we optionally maintain a `.Status.State` property with typed States on the respective resource.

We will not use upstream conditions. Independent structs allow us to specifically design clear conditions for our use-case.
We can maintain the structs, provide documentation, custom validation, etc. and don't risk potential breaking changes when upgrading the upstream library.
Lastly, decoupling our APIs from the upstream kubernetes libraries (as much as possible) wouldn't force others to use the same versions of the libraries we do. This makes it easier to consume the Greenhouse API.

### Conditions

The Greenhouse `Condition` has the following properties:

- `Type` (only one condition of a type may be applied to a resource at a point in time)
- `Status` (one of `True`, `False`, or `Unknown`)
- `LastTransitionTime` (Timestamp of the last transition)
- `Message` (human readable message to the last transition)

If it becomes necessary the condition can be expanded by a typed `Reason` for programmatic interaction.

Every reconcile step that needs to report success or failure back to the resource should return a custom condition instead of an error. All conditions are collected within the `StatusConditions`.
This struct provides a couple of convenience methods, so no direct interaction with the conditions array becomes necessary as it bears the risk to be error prone.

Every resource will maintain a greenhouse condition of the Type `Ready`. This will be the API endpoint to the resource overall "ready state" with respective message.
This `ReadyCondition` should only be computed by combining the other conditions. No other factors should be taken into consideration.

Each condition may only be manipulated and written by one controller. It may be read by various.

**Note**: `Condition.Status == false` does explicitely **not** mean failure of a reconciliation step. E.g. the `PodUnschedulable` Condition in k8s core.

Methods on the `StatusConditions`:

- `SetConditions(conditionsToSet ...Condition)`

  Updates an existing condition of matching `Type` with `LastTransitionTime` set to now if `Status` or `Message` differ, creates the condition otherwise.

- `GetConditionByType(conditionType ConditionType) *Condition`

  Returns a condition by it's `Type`

- `IsReadyTrue()`

  Returns `true` if the `Ready` Condition exists and it's `Status` is `true`

Methods on the `Condition`:

- `Equal(other Condition) bool`

  Compares two conditions on `Type`, `Status` and `Message`

- `IsTrue()`

  Returns `condition.Status == true`

We aim to provide helper methods and libraries for other clients to ease development and API interaction.

### Resource States

If we need to provide more differentiated resources states than only `Ready == true/false/unkown` we will introduce a typed `State` within the resource `Status`. This `State` should also be computed only by combining condition status.

Refer to the [plugin.Status.State](./../../pkg/apis/greenhouse/v1alpha1/pluginconfig_types.go#64) as a reference.

### HowTos and best practices

When reconciling an object we recommend to defer the reconciliation of the status within the `Reconcile()` method. Note how we pass the reference to the resource into the defer func:

```go
  var myResource = new(MyResource)
  if err := r.Get(ctx, req.NamespacedName, myResource); err != nil {
    return ctrl.Result{}, client.IgnoreNotFound(err)
  }
  ...

  defer func(myResource *MyResource) {
    if statusErr := r.reconcileStatus(ctx, myResource); statusErr != nil {
      log.FromContext(ctx).Error(statusErr, "failed to reconcile status")
    }
  }(myResource)

  ...
```

A reconciliation step in the `Reconcile()` method that should report back to the resource status is expected to return a condition (instead of an error), e.g.:

```go
  ...

  myResource.Status.SetConditions(r.someReconcileStep(ctx, myResource))

  ...
```

Following [SIG Architecture docs](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties)

> Controllers should apply their conditions to a resource the first time they visit the resource, even if the status is Unknown.

The `ReconcileStatus()` method should at least persist the all manipulated `conditions` back to the resource. Maybe it also computes the `ReadyCondition` or optionally a resource `State`. Note that not all controllers do, as they might reconcile different aspects of a resource.

```go
func (r *YourReconciler) reconcileStatus(ctx context.Context, myResource *MyResource) error {

  myCondition = myResource.Status.GetConditionByType("MyConditionType")
  if (myCondition == nil){
    mycondition = greenhousev1alpha1.Condition{
      Type:   "MyConditionType",
      Status: metav1.ConditionUnknown,
    }
  }
  ...

  _, err := clientutil.PatchStatus(ctx, r.Client, myResource, func() error {
    ...
    myResource.Status.SetConditions(myCondition, myOtherCondition)
    ...
    return nil
  })
  return err
}
```
