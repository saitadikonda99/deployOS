import { DashboardConfigError, type Device, fetchDevices } from "@/lib/devices";
import type { Metadata } from "next";
import { ContainerList } from "./container-list";
import { DeviceActions } from "./device-actions";

// This page always reflects live registration state, so it must never be
// statically cached across deploys - force per-request rendering
// regardless of whether fetchDevices() happens to reach its fetch() call
// during any given build (e.g. it won't if DEPLOYOS_API_URL/TOKEN are
// unset at build time, which would otherwise let Next statically bake in
// a stale result).
export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: "Devices - DeployOS Dashboard",
};

function formatOperatingSystem(os: string): string {
  return os.charAt(0).toUpperCase() + os.slice(1);
}

function formatRegistrationDate(isoDate: string): string {
  return new Date(isoDate).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function formatStatus(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function DevicesTable({ devices }: { devices: Device[] }) {
  if (devices.length === 0) {
    return (
      <p className="empty-state">
        No devices registered yet. Run the DeployOS agent on a machine to register it here.
      </p>
    );
  }

  return (
    <table>
      <thead>
        <tr>
          <th>Hostname</th>
          <th>Operating System</th>
          <th>Architecture</th>
          <th>Registration Date</th>
          <th>Status</th>
          <th>Actions</th>
          <th>Containers</th>
        </tr>
      </thead>
      <tbody>
        {devices.map((device) => {
          const connected = device.status === "connected";
          return (
            <tr key={device.id}>
              <td>{device.hostname}</td>
              <td>{formatOperatingSystem(device.operating_system)}</td>
              <td>{device.architecture}</td>
              <td>{formatRegistrationDate(device.created_at)}</td>
              <td>
                <span className={`status-badge${connected ? " status-badge--connected" : ""}`}>
                  {formatStatus(device.status)}
                </span>
              </td>
              <td>
                <DeviceActions deviceId={device.id} connected={connected} />
              </td>
              <td>
                <ContainerList deviceId={device.id} connected={connected} />
              </td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

export default async function DevicesPage() {
  let devices: Device[] = [];
  let errorMessage: string | null = null;

  try {
    devices = await fetchDevices();
  } catch (err) {
    errorMessage =
      err instanceof DashboardConfigError
        ? err.message
        : "Could not reach the DeployOS control plane. Check that it's running and reachable.";
  }

  return (
    <main>
      <h1>Devices</h1>
      <p className="subtitle">Machines registered to your DeployOS account.</p>

      {errorMessage ? (
        <p className="error-state">{errorMessage}</p>
      ) : (
        <DevicesTable devices={devices} />
      )}
    </main>
  );
}
