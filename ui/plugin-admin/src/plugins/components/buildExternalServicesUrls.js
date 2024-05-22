export const buildExternalServicesUrls = (exposedServices) => {
  // logs the stringified object

  if (!exposedServices) return null

  const links = []
  for (const url in exposedServices) {
    const currentObject = exposedServices[url]

    links.push({
      url: url,
      name: currentObject.name ? currentObject.name : url,
    })
  }

  return links
}
