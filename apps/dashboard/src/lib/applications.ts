export interface Application {
  id: string;
  name: string;
  status: string;
  runtime: string;
  created_at: string;
}

/**
 * Placeholder data for the Applications page. internal/applications
 * (see docs/application-engine.md) defines the Application domain
 * model and its lifecycle, but no HTTP API or persistence layer exists
 * yet - this previews the intended UI ahead of that work landing, per
 * this phase's scope (domain model and architecture only, no
 * deployment behavior).
 */
const MOCK_APPLICATIONS: Application[] = [
  {
    id: "9d2f7c1e-0a1b-4c3d-8e4f-1a2b3c4d5e6f",
    name: "marketing-site",
    status: "running",
    runtime: "docker",
    created_at: "2026-06-01T10:00:00Z",
  },
  {
    id: "1b3e5d7f-9a0c-4e2d-8b1a-3c4d5e6f7a8b",
    name: "api-gateway",
    status: "deploying",
    runtime: "docker",
    created_at: "2026-07-15T14:30:00Z",
  },
  {
    id: "4c6e8f0a-2b3d-4e5f-9a1b-5c6d7e8f9a0b",
    name: "worker-queue",
    status: "stopped",
    runtime: "docker",
    created_at: "2026-05-20T09:15:00Z",
  },
  {
    id: "7f9a1c3e-5b6d-4f7e-8a9b-7c8d9e0f1a2b",
    name: "docs-preview",
    status: "failed",
    runtime: "docker",
    created_at: "2026-07-28T18:45:00Z",
  },
];

/**
 * Returns the operator's applications. Currently always resolves to
 * placeholder data - see the MOCK_APPLICATIONS comment above. Kept as
 * an async function (rather than exporting the array directly) so the
 * eventual real implementation - an API call, following fetchDevices'
 * shape in lib/devices.ts - is a drop-in replacement for callers.
 */
export async function fetchApplications(): Promise<Application[]> {
  return MOCK_APPLICATIONS;
}
