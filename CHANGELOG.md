# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-01-03

Initial stable release.

### Features

- Automatic DNS A record management for Docker containers with Traefik labels
- Docker Swarm support with service label detection
- Real-time event watching for container start/stop events
- Startup reconciliation to sync existing containers
- Include/exclude regex patterns for hostname filtering
- Prometheus metrics endpoint with detailed operational metrics
- Health and readiness endpoints for orchestrator integration
- Docker secrets support via `_FILE` suffix on environment variables
- Dry run mode for testing configuration
- Multi-architecture Docker images (linux/amd64, linux/arm64)
- Docker socket proxy compatibility

### Configuration

- `TECHNITIUM_URL`: Technitium DNS server URL (required)
- `TECHNITIUM_TOKEN`: API token (required, supports `_FILE` suffix)
- `TECHNITIUM_ZONE`: DNS zone to manage (required)
- `TARGET_IP`: IP address for A records (required)
- `TTL`: Record TTL in seconds (default: 300)
- `INCLUDE_PATTERN`: Regex for hostnames to include (default: `.*`)
- `EXCLUDE_PATTERN`: Regex for hostnames to exclude
- `DOCKER_HOST`: Docker socket or TCP address
- `DOCKER_MODE`: `auto`, `swarm`, or `standalone`
- `RECONCILE_ON_STARTUP`: Run full sync at startup (default: true)
- `DRY_RUN`: Log changes without applying (default: false)
- `HEALTH_PORT`: Port for health/metrics endpoints (default: 8080)
- `LOG_LEVEL`: Logging verbosity (default: info)
