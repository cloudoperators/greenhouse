#!/usr/bin/env sh
# SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
# SPDX-License-Identifier: Apache-2.0


# extract args from the command line. Args are of the form --name=value
for i in "$@"; do
  case $i in
  --manifest=*)
    manifest="${i#*=}"
    shift # past argument=value
    ;;
  --apps=*)
    apps="${i#*=}"
    shift # past argument=value
    ;;
  --extensions=*)
    extensions="${i#*=}"
    shift # past argument=value
    ;;
  *)
    # unknown option
    ;;
  esac
done

# Assign default value if manifest is empty
manifest="${manifest:-manifest.json}"
# Assign default value if apps is empty
apps="${apps:-./apps}"
# Assign default value if manifest is empty
extensions="${extensions:-./extensions}"

# Function to convert semantic version to comparable integer value
convert_version() {
  echo "$1" | awk -F. '{printf("%d%03d%03d\n", $1, $2, $3)}'
}

echo '{}' | jq '.' >"$manifest"

for file in $(find "$apps" "$extensions" -name "package.json"); do
  name=$(jq -r '.name' "$file")
  main=$(jq -r '.main // .module' "$file")
  version=$(jq -r '.version' "$file")
  latest=$(jq --arg name "$name" -r '.[$name].latest.version // "0.0.0"' "$file")

  relative_path="${file#"$PWD"}"
  # Construct the path
  path="$(dirname "$relative_path")/$main"
  # Step 3: Add/extend the JSON file with the new structured entry
  jq --arg name "$name" --arg path "$path" --arg version "$version" \
    '.[$name] += { ($version): { name: $name, "path": $path, "version": $version } }' \
    "$manifest" >tmp.json && mv tmp.json "$manifest"

  last=$(convert_version $latest)
  current=$(convert_version $version)

  if [ "$last" -lt "$current" ]; then
    echo "$name: $latest is less than $version"
    jq --arg name "$name" --arg path "$path" --arg version "$version" \
      '.[$name] += { latest: { name: $name, "path": $path, "version": $version } }' \
      "$manifest" >tmp.json && mv tmp.json "$manifest"
  fi
done
