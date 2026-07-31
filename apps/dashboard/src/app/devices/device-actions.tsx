"use client";

import { type CommandResult, sendCommand } from "@/lib/commands";
import { useState } from "react";

const COMMANDS: { label: string; kind: string }[] = [
  { label: "Ping", kind: "PING" },
  { label: "Get Version", kind: "GET_VERSION" },
  { label: "Get Info", kind: "GET_INFO" },
];

export function DeviceActions({ deviceId, connected }: { deviceId: string; connected: boolean }) {
  const [pending, setPending] = useState<string | null>(null);
  const [result, setResult] = useState<CommandResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function runCommand(kind: string) {
    setPending(kind);
    setError(null);
    setResult(null);
    try {
      setResult(await sendCommand(deviceId, kind));
    } catch (err) {
      setError(err instanceof Error ? err.message : "command failed");
    } finally {
      setPending(null);
    }
  }

  return (
    <div className="device-actions">
      <div className="device-actions__buttons">
        {COMMANDS.map((command) => (
          <button
            key={command.kind}
            type="button"
            disabled={!connected || pending !== null}
            onClick={() => runCommand(command.kind)}
          >
            {pending === command.kind ? "Running..." : command.label}
          </button>
        ))}
      </div>

      {error ? (
        <p className="device-actions__result device-actions__result--error">{error}</p>
      ) : null}

      {result ? (
        <div
          className={`device-actions__result${result.success ? "" : " device-actions__result--error"}`}
        >
          <p>{result.message || (result.success ? "OK" : "Failed")}</p>
          {result.details ? (
            <dl>
              {Object.entries(result.details).map(([key, value]) => (
                <div key={key}>
                  <dt>{key}</dt>
                  <dd>{value}</dd>
                </div>
              ))}
            </dl>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
