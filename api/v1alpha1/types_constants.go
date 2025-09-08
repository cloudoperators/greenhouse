package v1alpha1

const (
	// GreenhouseOperation is the annotation used to trigger specific operations on a Greenhouse resource
	GreenhouseOperation string = "greenhouse.sap/operation"

	// GreenhouseOperationReconcile is the value used to trigger a reconcile operation on a Greenhouse resource
	GreenhouseOperationReconcile string = "reconcile"
)
