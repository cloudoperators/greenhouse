# Authorization Webhook E2E Tests

This directory contains end-to-end tests for the Greenhouse authorization webhook. The tests verify that the authorization webhook correctly allows or denies access to resources based on user support-group claims and resource ownership labels.

## Overview

The authorization webhook checks if a user has a `support-group:` claim that matches the `greenhouse.sap/owned-by` label on resources. If they match, the user gets full permissions on that resource.

## Test Structure

- `e2e_test.go` - Main test suite with BeforeSuite, AfterSuite, and test scenarios
- `fixtures/fixtures.go` - Helper functions for creating test resources
- `testdata/organization.yaml` - Organization resource for tests

## Test Scenarios

The test suite includes comprehensive coverage of authorization decisions:

### GET Operations

- ✅ Allow: Users with matching support-group can GET their owned resources
- ❌ Deny: Users without matching support-group cannot GET specific resources they don't own

**Note**: LIST operations are not supported by the authorization webhook. All operations require a specific resource name.

### UPDATE Operations

- ✅ Allow: Users with matching support-group can UPDATE their owned resources
- ❌ Deny: Users without matching support-group cannot UPDATE other resources

### DELETE Operations

- ✅ Allow: Users with matching support-group can DELETE their owned resources
- ❌ Deny: Users without matching support-group cannot DELETE other resources

## Running the Tests

### Prerequisites

1. Ensure you have the required tools installed:
   - kubectl
   - kind
   - go
   - openssl (for certificate generation)

### Setup and Run

1. **Setup the E2E environment with authorization webhook and certificates:**

   ```bash
   make setup-e2e-authz
   ```

2. **Run the authz E2E tests:**

   ```bash
   make e2e-local-authz
   ```

   Note: The `SCENARIO=authz` parameter is automatically set by the makefile target.

### Cleanup

To clean up the test environment:

```bash
make clean-e2e-authz
make clean-authz-certs
```

## Implementation Details

### Authorization Webhook Configuration

The test cluster uses a special configuration (`e2e/greenhouse-authz-cluster.yaml`) that:

- Mounts the webhook certificates and configuration
- Configures the Kubernetes API server to use the authorization webhook
- Uses structured authorization configuration

### User Impersonation

The tests use Kubernetes client-go's built-in impersonation feature rather than kubectl commands. The `createImpersonatedClient` helper function creates a REST config with impersonation settings:

```go
impersonatedConfig.Impersonate = rest.ImpersonationConfig{
    UserName: user,
    Groups:   groups,
}
```

This allows the tests to simulate requests from different users with different group memberships, testing the authorization webhook's behavior for various permission scenarios.

## Troubleshooting

### Check Authorization Webhook Logs

After running tests, check the generated controller logs:

```bash
cat bin/greenhouse-authz-e2e-pod-logs.txt
```
