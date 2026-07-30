import { loadConfig } from "./config.js";
import { createControlPlaneServer } from "./server.js";

const config = loadConfig();
const server = createControlPlaneServer();

server.listen(config.port, config.host, () => {
  console.info(`control-plane listening on http://${config.host}:${config.port}`);
});
