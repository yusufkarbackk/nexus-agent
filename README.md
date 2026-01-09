# Nexus Agent

A lightweight local proxy that enables **any programming language** to send encrypted data to Nexus. The agent handles AES-256-GCM encryption and forwards data securely to your Nexus server.

## Features

- **Language Agnostic**: Any language that can make HTTP requests can use the agent
- **Automatic Encryption**: AES-256-GCM encryption handled automatically
- **Offline Buffering**: SQLite-based queue for when Nexus server is unavailable
- **Multi-App Support**: Configure multiple sender apps in one agent
- **Auto-Retry**: Automatic retry with exponential backoff
- **Health Monitoring**: `/health` endpoint for monitoring

## Installation

### Option 1: Download Binary

Download the appropriate binary for your platform from the releases page.

### Option 2: Build from Source

```bash
cd nexus-agent
go build -o nexus-agent ./cmd/agent
```

### Option 3: Docker

```bash
docker build -t nexus-agent .
docker run -d \
  -p 9000:9000 \
  -v /path/to/config.yml:/etc/nexus-agent/config.yml \
  nexus-agent
```

## Configuration

Copy `config.example.yml` to `config.yml` and edit:

```yaml
agent:
  port: 9000
  bind: "127.0.0.1"  # Only allow local connections

nexus:
  server_url: "https://your-nexus-server.com"
  timeout: 30s
  retry_attempts: 3
  retry_delay: 5s

apps:
  - name: "Production App"
    app_key: "your_app_key"
    master_secret: "your_master_secret"

buffer:
  enabled: true
  max_size: 10000
  db_path: "./queue.db"
```

## Usage

### Start the Agent

```bash
./nexus-agent -config config.yml
```

### Send Data (Any Language)

Send a POST request to `http://localhost:9000/send`:

```json
{
  "app_key": "your_app_key",
  "data": {
    "temperature": 25.5,
    "humidity": 60,
    "device_id": "sensor-001"
  }
}
```

### Examples

**cURL:**
```bash
curl -X POST http://localhost:9000/send \
  -H "Content-Type: application/json" \
  -d '{"app_key": "your_app_key", "data": {"temp": 25}}'
```

**PHP:**
```php
$data = [
    'app_key' => 'your_app_key',
    'data' => ['temperature' => 25.5]
];

$ch = curl_init('http://localhost:9000/send');
curl_setopt($ch, CURLOPT_POST, true);
curl_setopt($ch, CURLOPT_POSTFIELDS, json_encode($data));
curl_setopt($ch, CURLOPT_HTTPHEADER, ['Content-Type: application/json']);
curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);
$response = curl_exec($ch);
```

**Go:**
```go
data := map[string]interface{}{
    "app_key": "your_app_key",
    "data": map[string]interface{}{
        "temperature": 25.5,
    },
}
jsonData, _ := json.Marshal(data)
http.Post("http://localhost:9000/send", "application/json", bytes.NewReader(jsonData))
```

**Java:**
```java
String json = "{\"app_key\": \"your_app_key\", \"data\": {\"temperature\": 25.5}}";
HttpRequest request = HttpRequest.newBuilder()
    .uri(URI.create("http://localhost:9000/send"))
    .POST(HttpRequest.BodyPublishers.ofString(json))
    .header("Content-Type", "application/json")
    .build();
HttpResponse<String> response = client.send(request, HttpResponse.BodyHandlers.ofString());
```

### Health Check

```bash
curl http://localhost:9000/health
```

Response:
```json
{
  "status": "healthy",
  "queue_size": 0,
  "apps_configured": 2
}
```

## Running as a Service

### Linux (systemd)

Create `/etc/systemd/system/nexus-agent.service`:

```ini
[Unit]
Description=Nexus Agent
After=network.target

[Service]
Type=simple
User=nexus
ExecStart=/usr/local/bin/nexus-agent -config /etc/nexus-agent/config.yml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Then:
```bash
sudo systemctl enable nexus-agent
sudo systemctl start nexus-agent
```

### Windows (as a Service)

Use NSSM or similar tool to run as a Windows service.

## Security

- The agent binds to `127.0.0.1` by default (localhost only)
- Master secrets are stored in the config file - secure file permissions recommended
- All communication to Nexus server uses HTTPS
- No sensitive data is logged
