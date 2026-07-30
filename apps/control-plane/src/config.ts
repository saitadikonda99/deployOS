export interface ControlPlaneConfig {
  port: number;
  host: string;
}

export function loadConfig(env: NodeJS.ProcessEnv = process.env): ControlPlaneConfig {
  const port = Number.parseInt(env.PORT ?? "4000", 10);

  if (Number.isNaN(port) || port <= 0 || port > 65535) {
    throw new Error(`Invalid PORT value: ${env.PORT}`);
  }

  return {
    port,
    host: env.HOST ?? "0.0.0.0",
  };
}
