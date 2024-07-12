

OUT_PATH="docs/apidocs"
TEMP_DIR="./tmp"

# check if temp directory exists
if [[ ! -d "${TEMP_DIR}" ]]; then
    mkdir -p "${TEMP_DIR}"
fi

# Check the doc-gen binary exists in the temp directory
DOC_GEN="${TEMP_DIR}/gen-crd-api-reference-docs"
if ! [[ -f "${DOC_GEN}" ]]; then
    echo "${DOC_GEN} does not exist. Installing..."
    cd "${TEMP_DIR}"
    GOBIN=$(pwd) go install github.com/ahmetb/gen-crd-api-reference-docs@latest
fi

GO111MODULE=on $DOC_GEN \
  -api-dir="../../pkg/apis/greenhouse/v1alpha1" \
  -config="./config.json" \
  -template-dir="./templates" \
  -out-file="../../docs/reference/api/index.html"
