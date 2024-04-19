export const useFetchMarkDown = () => {
  const fetchMarkDown = async (url: string): Promise<string> => {
    return fetch(url)
    .then((response) =>{
      if (!response.ok) {
        console.log(`failed fetching plugin readme from ${url}.`)
      }
      return response.text()})
      .catch((error) => {
        console.error(error)
        return ""
      })
  }

  return {
    fetchMarkDown: fetchMarkDown,
  }
}

export default useFetchMarkDown