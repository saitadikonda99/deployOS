"use client";

import { sendCommand } from "@/lib/commands";
import type { Container, ContainerDetails } from "@/lib/containers";
import { useState } from "react";

function formatCreated(isoDate: string): string {
  const date = new Date(isoDate);
  return Number.isNaN(date.getTime()) ? isoDate : date.toLocaleString();
}

export function ContainerList({ deviceId, connected }: { deviceId: string; connected: boolean }) {
  const [containers, setContainers] = useState<Container[] | null>(null);
  const [details, setDetails] = useState<ContainerDetails | null>(null);
  const [loading, setLoading] = useState(false);
  const [inspecting, setInspecting] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function loadContainers() {
    setLoading(true);
    setError(null);
    setDetails(null);
    try {
      const result = await sendCommand(deviceId, "LIST_CONTAINERS");
      if (!result.success) {
        throw new Error(result.message || "listing containers failed");
      }
      setContainers(result.details?.containers ? JSON.parse(result.details.containers) : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : "listing containers failed");
      setContainers(null);
    } finally {
      setLoading(false);
    }
  }

  async function inspect(id: string) {
    setInspecting(id);
    setError(null);
    try {
      const result = await sendCommand(deviceId, "INSPECT_CONTAINER", { id });
      if (!result.success) {
        throw new Error(result.message || "inspecting container failed");
      }
      setDetails(result.details?.container ? JSON.parse(result.details.container) : null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "inspecting container failed");
    } finally {
      setInspecting(null);
    }
  }

  return (
    <div className="container-list">
      <button type="button" disabled={!connected || loading} onClick={loadContainers}>
        {loading ? "Loading..." : "View Containers"}
      </button>

      {error ? (
        <p className="device-actions__result device-actions__result--error">{error}</p>
      ) : null}

      {containers ? (
        containers.length === 0 ? (
          <p className="empty-state">No containers found on this device.</p>
        ) : (
          <table className="container-list__table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Image</th>
                <th>Status</th>
                <th />
              </tr>
            </thead>
            <tbody>
              {containers.map((container) => (
                <tr key={container.id}>
                  <td>{container.name}</td>
                  <td>{container.image}</td>
                  <td>{container.status}</td>
                  <td>
                    <button
                      type="button"
                      disabled={inspecting !== null}
                      onClick={() => inspect(container.id)}
                    >
                      {inspecting === container.id ? "Inspecting..." : "Inspect"}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )
      ) : null}

      {details ? (
        <dl className="container-list__details">
          <div>
            <dt>Command</dt>
            <dd>{details.command.join(" ") || "-"}</dd>
          </div>
          <div>
            <dt>Created</dt>
            <dd>{formatCreated(details.created)}</dd>
          </div>
          <div>
            <dt>Restart Count</dt>
            <dd>{details.restart_count}</dd>
          </div>
          <div>
            <dt>Networks</dt>
            <dd>{details.networks.join(", ") || "none"}</dd>
          </div>
          <div>
            <dt>Ports</dt>
            <dd>
              {details.ports
                .map((p) => `${p.host_port}:${p.container_port}/${p.protocol}`)
                .join(", ") || "none"}
            </dd>
          </div>
          <div>
            <dt>Mounts</dt>
            <dd>
              {details.mounts.map((m) => `${m.source} -> ${m.destination}`).join(", ") || "none"}
            </dd>
          </div>
        </dl>
      ) : null}
    </div>
  );
}
