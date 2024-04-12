
import { Plugin } from "../../../../../types/types"


const SUCCESS_MESSAGE="Success!"


const postPlugin = async (plugin: Plugin, namespace:String, client:any):Promise<string> => {

  return await client
      .post(
        `/apis/greenhouse.sap/v1alpha1/namespaces/${namespace}/plugins/${
          plugin.metadata!.name
        }`,
        { ...plugin }
      )
      .then((res) => {
        console.log(res)
        return SUCCESS_MESSAGE
      })
      .catch((err) => {
        console.log(err)
         return err.message
      })
}

export default postPlugin
