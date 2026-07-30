import { type IncomingMessage, type ServerResponse, createServer } from "node:http";

interface RouteContext {
  req: IncomingMessage;
  res: ServerResponse;
}

type Route = (ctx: RouteContext) => void;

const routes: Record<string, Route> = {
  "GET /healthz": ({ res }) => {
    res.writeHead(200, { "content-type": "application/json" });
    res.end(JSON.stringify({ status: "ok" }));
  },
  "GET /version": ({ res }) => {
    res.writeHead(200, { "content-type": "application/json" });
    res.end(JSON.stringify({ name: "@deployos/control-plane", version: "0.0.0" }));
  },
};

export function createControlPlaneServer() {
  return createServer((req, res) => {
    const key = `${req.method} ${req.url}`;
    const route = routes[key];

    if (!route) {
      res.writeHead(404, { "content-type": "application/json" });
      res.end(JSON.stringify({ error: "not found" }));
      return;
    }

    route({ req, res });
  });
}
