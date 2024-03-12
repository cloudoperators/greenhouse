#!/usr/bin/env bash
# Copyright 2024-2026 SAP SE or an SAP affiliate company and Greenhouse contributors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


yq -e 'with(.components.schemas[]; .properties.metadata |= load("./scripts/templates/metadata.yaml"))' ../../docs/reference/api/openapi.yaml > ./temp && echo "successfully injected metadata into open api specs" || echo "failed injecting metadata into open api specs"
npx openapi-typescript ./temp -o ./src/types/schema.d.ts && echo "successfully generated typescript types" || echo "failed generating typescript types"
rm ./temp && echo "successfully removed temp file" || echo "failed removing temp file"
