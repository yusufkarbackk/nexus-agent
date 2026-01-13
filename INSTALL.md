# Nexus Agent Installation Guide

Download and install Nexus Agent to securely send data to your Nexus server.

---

## Quick Install (Linux/Ubuntu)

```bash
# Download latest version
wget https://github.com/YOUR_ORG/nexus-agent/releases/latest/download/nexus-agent-linux-amd64

# Make executable
chmod +x nexus-agent-linux-amd64

# Move to system path
sudo mv nexus-agent-linux-amd64 /usr/local/bin/nexus-agent

# Verify installation
nexus-agent --version
```

---

## Configuration

### 1. Create config file

```bash
sudo mkdir -p /etc/nexus
sudo nano /etc/nexus/config.yml
```

### 2. Add configuration (Auto-Sync mode)

```yaml
agent:
  port: 9000
  bind: "127.0.0.1"

nexus:
  server_url: "https://nexus.yourcompany.com"
  agent_token: "agt_YOUR_TOKEN_FROM_NEXUS_UI"
  sync_interval: 60s
  timeout: 30s
  retry_attempts: 3
  retry_delay: 5s

buffer:
  enabled: true
  max_size: 10000
  db_path: "/var/lib/nexus/queue.db"
```

### 3. Create data directory

```bash
sudo mkdir -p /var/lib/nexus
sudo chown $USER:$USER /var/lib/nexus
```

---

## Run as Service (Systemd)

### 1. Create service file

```bash
sudo nano /etc/systemd/system/nexus-agent.service
```

```ini
[Unit]
Description=Nexus Agent
After=network.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/nexus-agent -config /etc/nexus/config.yml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

### 2. Enable and start

```bash
sudo systemctl daemon-reload
sudo systemctl enable nexus-agent
sudo systemctl start nexus-agent
```

### 3. Check status

```bash
sudo systemctl status nexus-agent
sudo journalctl -u nexus-agent -f
```

---

## Test Agent

```bash
# Health check
curl http://localhost:9000/health

# Send test data
curl -X POST http://localhost:9000/send \
  -H "Content-Type: application/json" \
  -d '{"app_key": "app_xxx", "data": {"test": "hello"}}'
```

---

## Troubleshooting

| Issue | Solution |
|-------|----------|
| Permission denied | Run with `sudo` or fix file permissions |
| Connection refused | Check if agent is running: `systemctl status nexus-agent` |
| Sync failed | Verify `agent_token` and `server_url` in config |
| Buffered messages | Check `/var/lib/nexus/queue.db` for offline queue |

---

## Downloads

| Platform | Architecture | File |
|----------|--------------|------|
| Linux | AMD64 | `nexus-agent-linux-amd64` |
| Linux | ARM64 | `nexus-agent-linux-arm64` |
| Windows | AMD64 | `nexus-agent-windows-amd64.exe` |
| macOS | Intel | `nexus-agent-darwin-amd64` |
| macOS | Apple Silicon | `nexus-agent-darwin-arm64` |

Download from: [GitHub Releases](https://github.com/YOUR_ORG/nexus-agent/releases)
