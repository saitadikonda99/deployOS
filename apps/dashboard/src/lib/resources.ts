export interface Resource {
  id: string;
  name: string;
  type: string;
  status: string;
  /**
   * Names of the Applications that reference this Resource. Computed
   * by joining Application.ResourceRefs against Resource.ID - not a
   * field stored on the Resource itself, since internal/resources has
   * no idea internal/applications exists (see "Why resources are
   * independent from applications" in docs/resource-engine.md). Empty
   * means no Application currently depends on this Resource.
   */
  used_by: string[];
  created_at: string;
}

/**
 * Placeholder data for the Resources page. internal/resources (see
 * docs/resource-engine.md) defines the Resource domain model, its
 * type registry, and its lifecycle, but no HTTP API or persistence
 * layer exists yet - this previews the intended UI ahead of that work
 * landing, per this phase's scope (domain model and architecture only,
 * no provisioning behavior). used_by references the same placeholder
 * application names as lib/applications.ts's mock data.
 */
const MOCK_RESOURCES: Resource[] = [
  {
    id: "b1e2c3d4-5f6a-4b7c-8d9e-0f1a2b3c4d5e",
    name: "primary-db",
    type: "DATABASE",
    status: "available",
    used_by: ["marketing-site", "api-gateway"],
    created_at: "2026-05-28T09:00:00Z",
  },
  {
    id: "c2f3d4e5-6a7b-4c8d-9e0f-1a2b3c4d5e6f",
    name: "session-cache",
    type: "CACHE",
    status: "available",
    used_by: ["api-gateway"],
    created_at: "2026-06-10T11:20:00Z",
  },
  {
    id: "d3a4e5f6-7b8c-4d9e-0f1a-2b3c4d5e6f7a",
    name: "uploads-volume",
    type: "VOLUME",
    status: "provisioning",
    used_by: ["marketing-site"],
    created_at: "2026-07-30T16:05:00Z",
  },
  {
    id: "e4b5f6a7-8c9d-4e0f-1a2b-3c4d5e6f7a8b",
    name: "stripe-api-key",
    type: "SECRET",
    status: "available",
    used_by: ["api-gateway"],
    created_at: "2026-06-15T13:40:00Z",
  },
  {
    id: "f5c6a7b8-9d0e-4f1a-2b3c-4d5e6f7a8b9c",
    name: "app.example.com",
    type: "DOMAIN",
    status: "failed",
    used_by: ["marketing-site"],
    created_at: "2026-07-29T08:10:00Z",
  },
  {
    id: "a6d7b8c9-0e1f-4a2b-3c4d-5e6f7a8b9c0d",
    name: "build-cache",
    type: "CACHE",
    status: "pending",
    used_by: [],
    created_at: "2026-08-01T07:30:00Z",
  },
];

/**
 * Returns the operator's resources. Currently always resolves to
 * placeholder data - see the MOCK_RESOURCES comment above. Kept as an
 * async function (rather than exporting the array directly) so the
 * eventual real implementation - an API call, following fetchDevices'
 * shape in lib/devices.ts - is a drop-in replacement for callers.
 */
export async function fetchResources(): Promise<Resource[]> {
  return MOCK_RESOURCES;
}
