# CRD Documentation Tools

This folder contains tools and resources for generating CRD (Custom Resource Definitions) API reference documentation. It includes:

- The [gen-crd-api-reference-docs](https://github.com/ahmetb/gen-crd-api-reference-docs) binary for generating documentation.
- Templates and configuration files necessary for the documentation process.

## Usage

Run the following command to generate the CRD API reference documentation:

```bash
# Generate CRD API reference documentation
./hack/docs-generator/gen-crd-api-reference-docs -api-dir="./pkg/apis/greenhouse/v1alpha1" -config="./hack/docs-generator/config.json" -template-dir="./hack/docs-generator/templates" -out-file="./docs/reference/api/index.html"
```

Or, you can use the `make` target:

```bash
# Generate CRD API reference documentation
make generate-documentation
```
