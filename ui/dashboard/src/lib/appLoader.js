let manifest // Variable to cache the manifest data

export async function mount(
  containerElement,
  { name, version, assetsHost = document.location.origin, appProps = {} }
) {
  // Fetch the manifest only if it's not already cached
  if (!manifest) {
    const manifestUrl = new URL("/manifest.json", assetsHost)

    try {
      // Fetch and parse the manifest JSON
      manifest = await fetch(manifestUrl).then((response) => {
        if (!response.ok) {
          throw new Error(`Failed to fetch manifest: ${response.statusText}`)
        }
        return response.json()
      })
    } catch (error) {
      console.error("Error fetching manifest:", error)
      return false // Return false if the manifest couldn't be fetched
    }
  }

  // Check if the app name exists in the manifest
  if (!manifest[name]) {
    console.error(`No manifest found for ${name}`)
    return false // Return false if the app is not found in the manifest
  }

  // Get the path for the specified version or use "latest" (or entryFile to support legacy)
  const modulePath =
    manifest[name][version || "latest"]?.path ||
    manifest[name][version || "latest"]?.entryFile

  if (!modulePath) {
    console.error(`No path found for ${name} version ${version || "latest"}`)
    return false // Return false if the path is not found
  }

  const moduleUrl = new URL(modulePath, assetsHost)
  console.debug("===Load App:", name, version, "from", moduleUrl.href)

  try {
    // Dynamically import the module and extract its mount function
    const { mount } = await import(moduleUrl)

    // Mount the module with the provided container element and props
    await mount(containerElement, {
      props: appProps,
    })

    return true // Return true if the module was successfully mounted
  } catch (error) {
    console.error(`Error loading or mounting ${name}:`, error)
    return false // Return false if there was an error during import or mount
  }
}
