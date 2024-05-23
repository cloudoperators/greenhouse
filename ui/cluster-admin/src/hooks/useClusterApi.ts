import { Cluster } from "../../../types/types"
import useApi, { ApiResponse } from "./useApi"
import useNamespace from "./useNamespace"

export type ClusterApiResponse = {
  ok: boolean
  message: string
  response?: Cluster
}

export const useClusterApi = () => {
  const { get, create, update, deleteObject } = useApi()
  const { namespace } = useNamespace()

  const getCluster = (cluster: Cluster): Promise<ClusterApiResponse> => {
    return get<Cluster>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/clusters`,
      cluster
    ) as Promise<ClusterApiResponse>
  }

  const createCluster = (cluster: Cluster): Promise<ClusterApiResponse> => {
    return create<Cluster>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/clusters`,
      cluster
    ) as Promise<ClusterApiResponse>
  }

  const updateCluster = (cluster: Cluster): Promise<ClusterApiResponse> => {
    return update<Cluster>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/clusters`,
      cluster
    ) as Promise<ClusterApiResponse>
  }

  const deleteCluster = (cluster: Cluster): Promise<ClusterApiResponse> => {
    return deleteObject<Cluster>(
      `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/clusters`,
      cluster
    ) as Promise<ClusterApiResponse>
  }

  return { getCluster, createCluster, updateCluster, deleteCluster }
}
