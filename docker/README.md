# docker/

Container build and compose files for running DeployOS services will live
here as each service's runtime requirements stabilize:

- `docker/control-plane/Dockerfile`
- `docker/dashboard/Dockerfile`
- `docker/agent/Dockerfile`
- `docker-compose.yml` (local multi-service development stack)

None of these exist yet - the apps and crates in this repository are
scaffolding only, so there is nothing to containerize yet. Add them
alongside the first working version of each service, rather than shipping
speculative Dockerfiles ahead of real application code.
