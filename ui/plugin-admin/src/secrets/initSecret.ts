import { Secret } from "../../../types/types"

export const initSecret = (): Secret => {
  return {
    metadata: {
      name: "",
      namespace: "",
    },
    data: {},
  }
}
