export interface Container {
  id: string;
  name: string;
  image: string;
  status: string;
  state: string;
  created: string;
}

export interface Mount {
  source: string;
  destination: string;
  mode: string;
}

export interface Port {
  container_port: number;
  host_port: number;
  protocol: string;
}

export interface ContainerDetails extends Container {
  command: string[];
  env: string[];
  mounts: Mount[];
  ports: Port[];
  networks: string[];
  restart_count: number;
}
