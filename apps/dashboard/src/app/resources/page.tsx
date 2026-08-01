import { type Resource, fetchResources } from "@/lib/resources";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Resources - DeployOS Dashboard",
};

function formatStatus(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function formatType(type: string): string {
  return type.charAt(0) + type.slice(1).toLowerCase();
}

function formatCreatedDate(isoDate: string): string {
  return new Date(isoDate).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function formatUsedBy(usedBy: string[]): string {
  return usedBy.length > 0 ? usedBy.join(", ") : "Unused";
}

function statusBadgeClass(status: string): string {
  if (status === "available") {
    return "status-badge status-badge--connected";
  }
  if (status === "failed") {
    return "status-badge status-badge--failed";
  }
  return "status-badge";
}

function ResourcesTable({ resources }: { resources: Resource[] }) {
  if (resources.length === 0) {
    return (
      <p className="empty-state">
        No resources yet. Provisioning databases, caches, volumes, secrets, and domains is a future
        phase - see docs/resource-engine.md.
      </p>
    );
  }

  return (
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th>Type</th>
          <th>Status</th>
          <th>Used By</th>
          <th>Created At</th>
        </tr>
      </thead>
      <tbody>
        {resources.map((resource) => (
          <tr key={resource.id}>
            <td>{resource.name}</td>
            <td>{formatType(resource.type)}</td>
            <td>
              <span className={statusBadgeClass(resource.status)}>
                {formatStatus(resource.status)}
              </span>
            </td>
            <td>{formatUsedBy(resource.used_by)}</td>
            <td>{formatCreatedDate(resource.created_at)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default async function ResourcesPage() {
  const resources = await fetchResources();

  return (
    <main>
      <h1>Resources</h1>
      <p className="subtitle">
        Infrastructure resources managed by DeployOS. Placeholder data - see docs/resource-engine.md
        for what's implemented so far.
      </p>

      <ResourcesTable resources={resources} />
    </main>
  );
}
