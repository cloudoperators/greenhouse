#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
# SPDX-License-Identifier: Apache-2.0


yq -e 'with(.components.schemas[]; .properties.metadata |= load("./scripts/templates/metadata.yaml"))' ../../docs/reference/api/openapi.yaml > ./temp && echo "successfully injected metadata into open api specs" || echo "failed injecting metadata into open api specs"
npx openapi-typescript ./temp -o ./src/types/schema.d.ts && echo "successfully generated typescript types" || echo "failed generating typescript types"
rm ./temp && echo "successfully removed temp file" || echo "failed removing temp file"
