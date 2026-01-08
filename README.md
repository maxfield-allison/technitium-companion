# technitium-companion

> ## ⚠️ DEPRECATED — This project has been replaced by [dnsweaver](https://github.com/maxfield-allison/dnsweaver)
>
> **This repository will be removed on January 15, 2026.**
>
> ### Why the change?
>
> After releasing this project, I discovered that another project called [Technitium DNS Companion](https://github.com/Fail-Safe/Technitium-DNS-Companion) by [@Fail-Safe](https://github.com/Fail-Safe) already existed with essentially the same name. To avoid confusion in the Technitium community, I've rebranded and expanded this project into **dnsweaver**.
>
> ### What is dnsweaver?
>
> dnsweaver is the spiritual successor with expanded capabilities:
> - **Multiple DNS providers**: Technitium, Cloudflare (Route53, Pi-hole, AdGuard Home coming soon)
> - **Multiple sources**: Traefik labels + static config files (nginx, Caddy, HAProxy planned)
> - **Ownership tracking**: TXT records prevent accidental deletion of manual DNS entries
> - **Multi-provider routing**: Internal hostnames → Technitium, public hostnames → Cloudflare
>
> ### Migration
>
> dnsweaver is a drop-in replacement. Main config changes:
> - Environment prefix: `TC_` / `TECHNITIUM_` → `DNSWEAVER_`
> - Provider config is now named: `DNSWEAVER_{NAME}_TYPE=technitium`
>
> **Example migration:**
> ```bash
> # Old (technitium-companion)
> TECHNITIUM_URL=http://dns:5380
> TECHNITIUM_TOKEN=xxx
> TECHNITIUM_ZONE=home.example.com
> TARGET_IP=192.168.1.100
>
> # New (dnsweaver)
> DNSWEAVER_TECHNITIUM_TYPE=technitium
> DNSWEAVER_TECHNITIUM_URL=http://dns:5380
> DNSWEAVER_TECHNITIUM_TOKEN=xxx
> DNSWEAVER_TECHNITIUM_ZONE=home.example.com
> DNSWEAVER_TECHNITIUM_TARGET=192.168.1.100
> DNSWEAVER_TECHNITIUM_DOMAINS=*.home.example.com
> DNSWEAVER_PROVIDERS=technitium
> ```
>
> ### Links
>
> - **dnsweaver** (successor): https://github.com/maxfield-allison/dnsweaver
> - **Technitium DNS Companion** (the "official" holder of the companion name): https://github.com/Fail-Safe/Technitium-DNS-Companion
>
> ---

[![Release](https://img.shields.io/github/v/release/maxfield-allison/technitium-companion?style=flat-square)](https://github.com/maxfield-allison/technitium-companion/releases)
[![License](https://img.shields.io/github/license/maxfield-allison/technitium-companion?style=flat-square)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/maxfield-allison/technitium-companion?style=flat-square)](go.mod)

**⚠️ DEPRECATED — See notice above**

~~**Automatic DNS record management for Docker containers.**~~

~~technitium-companion watches Docker events and automatically creates and deletes DNS A records in [Technitium DNS Server](https://technitium.com/dns/) for services with Traefik labels. Built for homelabs and self-hosted environments where you want automatic DNS resolution for your containerized services.~~

## Features

- **Docker and Swarm Support**: Works with standalone Docker and Docker Swarm clusters
- **Traefik Integration**: Parses `traefik.http.routers.*.rule` labels to extract hostnames
- **Real-time Sync**: Watches Docker events and creates/deletes records instantly
- **Startup Reconciliation**: Full sync on startup ensures consistency
- **Flexible Filtering**: Include/exclude patterns to control which hostnames are managed
- **Prometheus Metrics**: Full observability with detailed metrics
- **Secrets Support**: Docker secrets compatible via `_FILE` suffix variables
- **Health Endpoints**: `/health`, `/ready`, and `/metrics` for monitoring
- **Socket Proxy Support**: Works with Docker socket proxies for enhanced security
- **Multi-arch Images**: Supports linux/amd64 and linux/arm64

## Installation

### Docker Hub

```bash
docker pull maxamill/technitium-companion:latest
```

### GitHub Container Registry

```bash
docker pull ghcr.io/maxfield-allison/technitium-companion:latest
```

## Quick Start

### Docker Compose

```yaml
services:
  technitium-companion:
    image: maxamill/technitium-companion:latest
    restart: unless-stopped
    environment:
      - TECHNITIUM_URL=http://your-technitium-server:5380
      - TECHNITIUM_TOKEN=your-api-token
      - TECHNITIUM_ZONE=home.example.com
      - TARGET_IP=192.168.1.100
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - "8080:8080"
```

### How It Works

1. A container starts with a Traefik label:
   ```yaml
   labels:
     - "traefik.http.routers.myapp.rule=Host(`myapp.home.example.com`)"
   ```

2. technitium-companion detects the event and creates an A record:
   ```
   myapp.home.example.com -> 192.168.1.100
   ```

3. When the container stops, the A record is automatically deleted.

## Configuration

All configuration is via environment variables. Variables support the `_FILE` suffix for Docker secrets (e.g., `TECHNITIUM_TOKEN_FILE=/run/secrets/dns_token`).

### Required Variables

| Variable | Description |
|----------|-------------|
| `TECHNITIUM_URL` | Technitium DNS server URL (e.g., `http://dns.example.com:5380`) |
| `TECHNITIUM_TOKEN` | API token from Technitium Admin, Settings, API |
| `TECHNITIUM_ZONE` | DNS zone to manage (e.g., `home.example.com`) |
| `TARGET_IP` | IP address for all A records (typically your ingress or load balancer) |

### Optional Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `TTL` | `300` | DNS record TTL in seconds |
| `INCLUDE_PATTERN` | `.*` | Regex pattern; only matching hostnames are managed |
| `EXCLUDE_PATTERN` | (none) | Regex pattern; matching hostnames are skipped |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker daemon socket or TCP address |
| `DOCKER_MODE` | `auto` | `auto` (detect), `swarm`, or `standalone` |
| `RECONCILE_ON_STARTUP` | `true` | Run full reconciliation at startup |
| `DRY_RUN` | `false` | Log changes without applying them |
| `HEALTH_PORT` | `8080` | Port for health and metrics endpoints |
| `LOG_LEVEL` | `info` | Logging level: `debug`, `info`, `warn`, `error` |

### Pattern Examples

```bash
# Only manage *.internal.example.com hostnames
INCLUDE_PATTERN='^[^.]+\.internal\.example\.com$'

# Exclude monitoring and test services
EXCLUDE_PATTERN='^(grafana|prometheus|test-).*'

# Combine both: only internal, but skip monitoring
INCLUDE_PATTERN='\.internal\.example\.com$'
EXCLUDE_PATTERN='^(grafana|prometheus)\.'
```

## Deployment

### Docker Compose (Standalone)

```yaml
services:
  technitium-companion:
    image: ghcr.io/maxfield-allison/technitium-companion:latest
    container_name: technitium-companion
    restart: unless-stopped
    environment:
      - TECHNITIUM_URL=http://technitium:5380
      - TECHNITIUM_TOKEN=${TECHNITIUM_TOKEN}
      - TECHNITIUM_ZONE=home.example.com
      - TARGET_IP=192.168.1.100
      - LOG_LEVEL=info
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    ports:
      - "8080:8080"
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
```

### Docker Swarm

For Swarm deployments, run on manager nodes to access Swarm service information:

```yaml
version: "3.8"

services:
  technitium-companion:
    image: ghcr.io/maxfield-allison/technitium-companion:latest
    deploy:
      mode: replicated
      replicas: 1
      placement:
        constraints:
          - node.role == manager
      restart_policy:
        condition: on-failure
        delay: 5s
        max_attempts: 3
    environment:
      - TECHNITIUM_URL=http://technitium:5380
      - TECHNITIUM_TOKEN_FILE=/run/secrets/technitium_token
      - TECHNITIUM_ZONE=home.example.com
      - TARGET_IP=192.168.1.100
      - DOCKER_MODE=swarm
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    secrets:
      - technitium_token
    networks:
      - internal
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3

secrets:
  technitium_token:
    external: true

networks:
  internal:
    external: true
```

Deploy with:
```bash
docker stack deploy -c docker-stack.yml technitium-companion
```

### Using a Docker Socket Proxy

For enhanced security, you can use a Docker socket proxy instead of mounting the Docker socket directly. This limits what API calls technitium-companion can make.

Example with [Tecnativa/docker-socket-proxy](https://github.com/Tecnativa/docker-socket-proxy):

```yaml
services:
  socket-proxy:
    image: tecnativa/docker-socket-proxy:latest
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
    environment:
      - CONTAINERS=1
      - SERVICES=1
      - TASKS=1
      - NETWORKS=1
      - EVENTS=1
    networks:
      - socket-proxy

  technitium-companion:
    image: ghcr.io/maxfield-allison/technitium-companion:latest
    environment:
      - DOCKER_HOST=tcp://socket-proxy:2375
      - TECHNITIUM_URL=http://technitium:5380
      - TECHNITIUM_TOKEN=${TECHNITIUM_TOKEN}
      - TECHNITIUM_ZONE=home.example.com
      - TARGET_IP=192.168.1.100
    networks:
      - socket-proxy
    depends_on:
      - socket-proxy

networks:
  socket-proxy:
    driver: bridge
```

Required socket proxy permissions:
- `CONTAINERS=1`: Read container information and labels
- `SERVICES=1`: Read Swarm service information (Swarm mode only)
- `TASKS=1`: Read Swarm task information (Swarm mode only)
- `NETWORKS=1`: Read network information
- `EVENTS=1`: Subscribe to Docker events

## Monitoring

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `/health` | Liveness probe; returns 200 if service is running |
| `/ready` | Readiness probe; returns 200 after startup reconciliation |
| `/metrics` | Prometheus metrics endpoint |

### Prometheus Metrics

Counters:
- `technitium_companion_dns_records_created_total{zone}`: DNS records created
- `technitium_companion_dns_records_deleted_total{zone}`: DNS records deleted
- `technitium_companion_dns_records_existed_total{zone}`: Records that already existed
- `technitium_companion_api_requests_total{endpoint,status}`: Technitium API calls
- `technitium_companion_docker_events_total{type,action}`: Docker events processed
- `technitium_companion_reconciliations_total{status}`: Reconciliation runs

Histograms:
- `technitium_companion_api_request_duration_seconds{endpoint}`: API latency
- `technitium_companion_reconciliation_duration_seconds`: Reconciliation duration

Gauges:
- `technitium_companion_up`: Service health (1 = up)
- `technitium_companion_workloads_scanned`: Workloads in last reconciliation
- `technitium_companion_hostnames_found`: Hostnames found in last reconciliation
- `technitium_companion_last_reconciliation_timestamp_seconds`: Last successful reconciliation
- `technitium_companion_build_info{version,go_version}`: Build information

### Grafana Dashboard

Example queries for a Grafana dashboard:

```promql
# DNS record creation rate (per minute)
rate(technitium_companion_dns_records_created_total[5m]) * 60

# API success rate
sum(rate(technitium_companion_api_requests_total{status="success"}[5m])) /
sum(rate(technitium_companion_api_requests_total[5m])) * 100

# Reconciliation success rate
sum(rate(technitium_companion_reconciliations_total{status="success"}[1h])) /
sum(rate(technitium_companion_reconciliations_total[1h])) * 100

# Time since last reconciliation
time() - technitium_companion_last_reconciliation_timestamp_seconds
```

### Prometheus Scrape Config

```yaml
scrape_configs:
  - job_name: 'technitium-companion'
    static_configs:
      - targets: ['technitium-companion:8080']
    scrape_interval: 15s
```

## Troubleshooting

### Common Issues

**No DNS records being created:**
1. Verify Traefik labels on your containers:
   ```bash
   docker inspect <container> | grep -i traefik
   ```
2. Check the label format. The Host value must be wrapped in backticks:
   ```
   traefik.http.routers.<name>.rule=Host(`hostname.example.com`)
   ```
3. Enable debug logging: `LOG_LEVEL=debug`
4. Check if hostname matches `INCLUDE_PATTERN` and does not match `EXCLUDE_PATTERN`

**Connection refused to Technitium:**
1. Verify `TECHNITIUM_URL` is reachable from the container
2. Check API token permissions in Technitium Admin under Settings, API
3. If running in Docker, ensure network connectivity to the DNS server

**Records created but not resolving:**
1. Verify the zone exists in Technitium
2. Check that `TARGET_IP` is correct
3. Ensure the zone is authoritative in Technitium

**Swarm mode not detecting services:**
1. Ensure container runs on a manager node (constraint: `node.role == manager`)
2. Set `DOCKER_MODE=swarm` explicitly
3. Verify Docker socket is mounted: `/var/run/docker.sock:/var/run/docker.sock:ro`

**Using a socket proxy and getting permission errors:**
1. Ensure the proxy has `EVENTS=1` enabled for event watching
2. For Swarm mode, enable `SERVICES=1` and `TASKS=1`
3. Check proxy logs for denied API calls

### Debug Mode

Enable verbose logging to troubleshoot:

```bash
docker run -e LOG_LEVEL=debug ...
```

Logs include:
- Docker events received
- Hostnames parsed from labels
- DNS API calls and responses
- Record create/delete operations

### Dry Run Mode

Test configuration without making changes:

```bash
docker run -e DRY_RUN=true ...
```

## Building from Source

```bash
# Clone
git clone https://github.com/maxfield-allison/technitium-companion.git
cd technitium-companion

# Build
go build -o technitium-companion ./cmd/technitium-companion

# Run tests
go test -v ./...

# Build Docker image
docker build -t technitium-companion:local .
```

## Contributing

Contributions are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Technitium DNS Server](https://technitium.com/dns/)
- [Traefik](https://traefik.io/)
- [Prometheus](https://prometheus.io/)
