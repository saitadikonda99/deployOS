export interface Device {
  id: string;
  hostname: string;
  operating_system: string;
  architecture: string;
  /** Live connection state - "connected" or "disconnected" - from the
   * control plane's in-memory Connection Manager, not a stored field. */
  status: string;
  created_at: string;
}

interface ListDevicesResponse {
  devices: Device[];
}

/**
 * Thrown when the dashboard is missing the server-side configuration it
 * needs to call the DeployOS API. Distinguished from a request failure so
 * the page can show an actionable message instead of a generic error.
 */
export class DashboardConfigError extends Error {}

/**
 * Fetches the authenticated operator's devices from the DeployOS control
 * plane. DEPLOYOS_API_TOKEN is a server-only secret (never prefixed with
 * NEXT_PUBLIC_) standing in for real operator login, which the dashboard
 * doesn't implement yet - see docs/device-registration.md.
 */
export async function fetchDevices(): Promise<Device[]> {
  const apiUrl = process.env.DEPLOYOS_API_URL;
  const apiToken = process.env.DEPLOYOS_API_TOKEN;

  if (!apiUrl || !apiToken) {
    throw new DashboardConfigError(
      "DEPLOYOS_API_URL and DEPLOYOS_API_TOKEN must be set to load devices.",
    );
  }

  const res = await fetch(`${apiUrl}/api/v1/devices`, {
    headers: { Authorization: `Bearer ${apiToken}` },
    cache: "no-store",
  });

  if (!res.ok) {
    throw new Error(`failed to load devices: control plane returned ${res.status}`);
  }

  const body: ListDevicesResponse = await res.json();
  return body.devices;
}
