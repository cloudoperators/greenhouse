import { useEffect } from "react";
import useUrlState from "../hooks/useUrlState";
import useWatch from "../hooks/useWatch";

interface AsyncWorkerProps {
  consumerId: string;
}

const AsyncWorker: React.FC<AsyncWorkerProps> = (props: AsyncWorkerProps) => {
  useUrlState(props.consumerId);

  const { watchPlugins: watchPlugins } = useWatch();

  useEffect(() => {
    if (!watchPlugins) return;
    const unwatch = watchPlugins();
    return unwatch;
  }, [watchPlugins]);

  return null;
};

export default AsyncWorker;
