import type { Metadata } from "next";
import Link from "next/link";
import type { ReactNode } from "react";
import "./globals.css";

export const metadata: Metadata = {
  title: "DeployOS Dashboard",
  description: "Manage deployments, secrets, and infrastructure across your DeployOS fleet.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="en">
      <body>
        <nav>
          <Link href="/">Home</Link>
          <Link href="/devices">Devices</Link>
          <Link href="/applications">Applications</Link>
        </nav>
        {children}
      </body>
    </html>
  );
}
