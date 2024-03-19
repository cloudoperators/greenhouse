import { Container } from "juno-ui-components";
import PluginDetail from "./components/PluginDetail";
import PluginGrid from "./components/PluginGrid";
import WelcomeView from "./components/WelcomeView";
import useStore from "./store";

const AppContent = () => {
  const plugins = useStore((state) => state.plugins);
  const showPluginDetails = useStore((state) => state.showPluginDetails);
  const pluginDetail = useStore((state) => state.pluginDetail);
  const auth = useStore((state) => state.auth);
  const authError = auth?.error;
  const loggedIn = useStore((state) => state.loggedIn);

  return (
    <Container>
      {loggedIn && !authError ? (
        <>
          {plugins.length > 0 && <PluginGrid plugins={plugins} />}
          {showPluginDetails && pluginDetail && (
            <PluginDetail plugin={pluginDetail} />
          )}
        </>
      ) : (
        <WelcomeView />
      )}
    </Container>
  );
};

export default AppContent;
