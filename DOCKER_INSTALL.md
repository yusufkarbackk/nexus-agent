# Nexus Agent - Docker Installation Guide

Quick guide for installing Nexus Agent using Docker.

---

## Prerequisites

- Docker installed
- Docker Compose (optional, but recommended)
- Agent Token from Nexus Dashboard

---

## Quick Start

### 1. Create directory

```bash
mkdir -p /opt/nexus-agent
cd /opt/nexus-agent
```

### 2. Create config.yml

```bash
cat > config.yml << 'EOF'
agent:
  port: 9000
  bind: "0.0.0.0"

nexus:
  server_url: "https://nexus.yourcompany.com"
  agent_token: "agt_YOUR_TOKEN_HERE"
  sync_interval: 60s
  timeout: 30s

buffer:
  enabled: true
  max_size: 10000
  db_path: "/var/lib/nexus/queue.db"
EOF
```

**Update:**
- `server_url` → Your Nexus server URL
- `agent_token` → Token from Nexus UI (Agents page)

### 3. Run with Docker

```bash
docker run -d \
  --name nexus-agent \
  -p 9000:9000 \
  -v $(pwd)/config.yml:/etc/nexus-agent/config.yml:ro \
  -v nexus-agent-data:/var/lib/nexus \
  --restart unless-stopped \
  ghcr.io/YOUR_ORG/nexus-agent:latest
```

### 4. Verify

```bash
# Check status
docker ps | grep nexus-agent

# Check logs
docker logs nexus-agent

# Test health
curl http://localhost:9000/health
```

---

## Using Docker Compose (Recommended)

### 1. Download files

```bash
mkdir -p /opt/nexus-agent
cd /opt/nexus-agent

# Download docker-compose
curl -O https://raw.githubusercontent.com/YOUR_ORG/nexus-agent/main/docker-compose.customer.yml
mv docker-compose.customer.yml docker-compose.yml

# Download config example
curl -O https://raw.githubusercontent.com/YOUR_ORG/nexus-agent/main/config.example.yml
mv config.example.yml config.yml
```

### 2. Edit config.yml

```bash
nano config.yml
```

Update `server_url` and `agent_token`.

### 3. Start

```bash
docker compose up -d
```

### 4. Manage

```bash
# View logs
docker compose logs -f

# Restart
docker compose restart

# Stop
docker compose down

# Update to latest version
docker compose pull
docker compose up -d
```

---

## Sending Data

### Health Check

```bash
curl http://localhost:9000/health
```

### Send Data

```bash
curl -X POST http://localhost:9000/send \
  -H "Content-Type: application/json" \
  -d '{
    "app_key": "app_YOUR_APP_KEY",
    "data": {
      "temperature": 25.5,
      "humidity": 60,
      "device_id": "sensor-001"
    }
  }'
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Connection refused | Check if container is running: `docker ps` |
| Sync failed | Verify `agent_token` and `server_url` in config |
| Permission denied | Check config.yml file permissions |
| Container keeps restarting | Check logs: `docker logs nexus-agent` |

---

## Firewall

If using UFW:
```bash
sudo ufw allow 9000/tcp
```

---

## Updating

```bash
cd /opt/nexus-agent
docker compose pull
docker compose up -d
```
