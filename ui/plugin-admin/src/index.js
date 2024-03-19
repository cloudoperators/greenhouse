import React from "react"
import { createRoot } from "react-dom/client"

// export mount and unmount functions with async import
export const mount = async (container, options = {}) => {
  await import("./App").then((App) => {
    mount.root = createRoot(container)
    mount.root.render(React.createElement(App.default, options?.props))
  })
}

export const unmount = () => mount.root && mount.root.unmount()
