import { type Application, fetchApplications } from "@/lib/applications";
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Applications - DeployOS Dashboard",
};

function formatStatus(status: string): string {
  return status.charAt(0).toUpperCase() + status.slice(1);
}

function formatRuntime(runtime: string): string {
  return runtime.charAt(0).toUpperCase() + runtime.slice(1);
}

function formatCreatedDate(isoDate: string): string {
  return new Date(isoDate).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}

function statusBadgeClass(status: string): string {
  if (status === "running") {
    return "status-badge status-badge--connected";
  }
  if (status === "failed") {
    return "status-badge status-badge--failed";
  }
  return "status-badge";
}

function ApplicationsTable({ applications }: { applications: Application[] }) {
  if (applications.length === 0) {
    return (
      <p className="empty-state">
        No applications yet. Deploying from Git is a future phase - see docs/application-engine.md.
      </p>
    );
  }

  return (
    <table>
      <thead>
        <tr>
          <th>Name</th>
          <th>Status</th>
          <th>Runtime</th>
          <th>Created Date</th>
        </tr>
      </thead>
      <tbody>
        {applications.map((app) => (
          <tr key={app.id}>
            <td>{app.name}</td>
            <td>
              <span className={statusBadgeClass(app.status)}>{formatStatus(app.status)}</span>
            </td>
            <td>{formatRuntime(app.runtime)}</td>
            <td>{formatCreatedDate(app.created_at)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

export default async function ApplicationsPage() {
  const applications = await fetchApplications();

  return (
    <main>
      <h1>Applications</h1>
      <p className="subtitle">
        Applications managed by DeployOS. Placeholder data - see docs/application-engine.md for
        what's implemented so far.
      </p>

      <ApplicationsTable applications={applications} />
    </main>
  );
}
