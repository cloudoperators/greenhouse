import { Secret } from "../../../../types/types"
import useApi, { ApiResponse } from "./useApi"
import useNamespace from "./useNamespace"


export const useSecretApi = () => {
  const {get, create,  update, deleteObject } = useApi()
  const { namespace } = useNamespace()

  const getSecret= (secret: Secret): Promise<ApiResponse> => {
    return get<Secret>(`/api/v1/namespaces/${namespace}/secrets`, secret)
  }

  const createSecret = (secret: Secret): Promise<ApiResponse> => {
    return create<Secret>(`/api/v1/namespaces/${namespace}/secrets`, secret)
  }

  const updateSecret = (secret: Secret): Promise<ApiResponse> => {
    return update<Secret>(`/api/v1/namespaces/${namespace}/secrets`, secret)
  }

  const deleteSecret = (secret: Secret): Promise<ApiResponse> => {
    return deleteObject <Secret>(`/api/v1/namespaces/${namespace}/secrets`, secret)
  }

  

  return { getSecret, createSecret, updateSecret, deleteSecret }
}

export default useSecretApi

