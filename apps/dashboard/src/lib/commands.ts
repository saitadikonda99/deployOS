"use server";

export interface CommandResult {
  command_id: string;
  success: boolean;
  message: string;
  details?: Record<string, string>;
}

/**
 * Sends a command to a device via the DeployOS control plane's Command
 * Bus and returns its structured result. A Server Action (not a client-
 * side fetch) so DEPLOYOS_API_TOKEN - a server-only secret standing in
 * for real operator login, see docs/device-registration.md - never
 * reaches the browser.
 */
export async function sendCommand(
  deviceId: string,
  kind: string,
  args?: Record<string, string>,
): Promise<CommandResult> {
  const apiUrl = process.env.DEPLOYOS_API_URL;
  const apiToken = process.env.DEPLOYOS_API_TOKEN;

  if (!apiUrl || !apiToken) {
    throw new Error("DEPLOYOS_API_URL and DEPLOYOS_API_TOKEN must be set to send commands.");
  }

  const res = await fetch(`${apiUrl}/api/v1/devices/${encodeURIComponent(deviceId)}/commands`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${apiToken}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ kind, arguments: args }),
    cache: "no-store",
  });

  if (!res.ok) {
    const body: { error?: string } = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `command failed: control plane returned ${res.status}`);
  }

  return res.json();
}
