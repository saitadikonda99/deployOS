# Security Policy

DeployOS turns a machine into a production-facing server, so security
issues are treated as a priority. If you believe you've found a
vulnerability, please help us by reporting it responsibly.

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, report it privately using one of the following:

- [GitHub Security Advisories](https://github.com/saitadikonda99/deployOS/security/advisories/new)
  for this repository (preferred), or
- Email **saitadikonda.cse@gmail.com** with a description of the issue.

Please include as much of the following as you can:

- A description of the vulnerability and its potential impact.
- Steps to reproduce, or a proof-of-concept.
- The affected component (`apps/dashboard`, `cmd/server`, `cmd/agent`,
  etc.) and version/commit.
- Any known mitigations.

## What to expect

- **Acknowledgement** within 3 business days of your report.
- **Initial assessment** (severity and affected versions) within 7 days.
- **Coordinated disclosure**: we will work with you on a disclosure
  timeline and credit you in the advisory, unless you prefer to remain
  anonymous.

## Supported versions

DeployOS is pre-1.0 and does not yet have a stable release line. Security
fixes are applied to the `main` branch; once tagged releases begin, this
section will list which versions receive fixes.

## Scope

This policy covers the code in this repository (`apps/`, `cmd/`,
`internal/`, `pkg/`). Vulnerabilities in upstream dependencies should be reported
to the respective upstream project, though we appreciate a heads-up so we
can track and update our own dependency.
