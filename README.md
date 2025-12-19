# 🚢 Harbor Lighthouse

**The Universal Telemetry Agent for [Harbor Scale](https://harborscale.com)**

Lighthouse is a single-binary, cross-platform agent designed to collect metrics from anywhere (Linux servers, Windows desktops, Meshtastic nodes, Python scripts) and ship them securely to Harbor Scale.

It features a hybrid engine that supports both high-volume "Cargo" metrics (CPU/RAM batching) and raw object streams (GPS/LoRaWAN), all while handling rate limits, auto-updates, and service management automatically.

---

## ⚡ Quick Start

### Linux / macOS / Raspberry Pi

```bash
# Install & Start Service
curl -sL [https://downloads.harborscale.com/install.sh](https://downloads.harborscale.com/install.sh) | sudo bash

# Add a Monitor (e.g., Linux System Stats)
lighthouse --add \
  --name "server-01" \
  --harbor-id "786" \
  --key "hs_live_xxxxxxxx" \
  --source linux

```

### Windows (PowerShell)

```powershell
# Download .exe from Releases, then run:
.\lighthouse.exe --install

# Add a Monitor
.\lighthouse.exe --add --name "office-pc" --harbor-id "786" --key "hs_live_xxxx" --source windows

```

---

## 🛠 Power User Guide

### 1. The Architecture

Lighthouse uses a **Fan-Out / Hybrid Architecture**:

- **Collector:** Gathers data (e.g., reads 50 system metrics).
- **Engine:** Checks the `Harbor Type`:
- **`general` (Cargo Mode):** Explodes the 50 metrics into individual standardized payloads and sends them via the **Batch API** (`/batch`).
- **`gps` (Raw Mode):** Injects metadata (Ship ID, Time) and sends the raw JSON object to the specialized endpoint (`/gps`).

- **Transport:** Handles retries, HTTP 429 backoff, and 413 Payload size errors.

### 2. Configuration (`lighthouse_config.json`)

Located next to the binary. You can edit this manually if you prefer not to use the CLI.

```json
{
  "auto_update": true,
  "instances": [
    {
      "name": "production-db",
      "harbor_id": "786",
      "api_key": "hs_live_key",
      "source": "linux",
      "harbor_type": "general",
      "interval": 60,
      "max_batch_size": 100,
      "params": {}
    },
    {
      "name": "roof-node",
      "harbor_id": "786",
      "api_key": "hs_live_key",
      "source": "meshtastic",
      "harbor_type": "gps",
      "interval": 300,
      "max_batch_size": 1,
      "params": {
        "ip": "192.168.1.50"
      }
    }
  ]
}
```

### 3. CLI Commands Reference

| Flag           | Description                           | Example                       |
| -------------- | ------------------------------------- | ----------------------------- |
| `--add`        | Registers a new monitoring task.      | `--add --name "x" ...`        |
| `--remove`     | Stops and deletes a task by name.     | `--remove "x"`                |
| `--list`       | Shows health status of all tasks.     | `--list`                      |
| `--logs`       | filters logs for a specific task.     | `--logs "x"`                  |
| `--install`    | Installs systemd/LaunchAgent/Service. | `sudo ./lighthouse --install` |
| `--autoupdate` | Toggles self-updating mechanism.      | `--autoupdate=false`          |

### 4. Advanced: Rate Limiting & Batching

To optimize for specific Harbor Scale plans, you can tune the collection engine per instance:

**High Frequency (Pro Plan):**

```bash
./lighthouse --add --name "high-freq" ... --interval 5 --batch-size 500

```

- **Effect:** Collects every 5s. If data > 500 items, it splits into multiple HTTP requests automatically.

**Low Bandwidth (IoT Plan):**

```bash
./lighthouse --add --name "iot-device" ... --interval 300 --batch-size 10

```

### 5. Debugging

If a specific instance is failing, use the instance-scoped log filter:

```bash
# See why 'roof-node' is offline
./lighthouse --logs "roof-node"

```

_Output:_

```text
[roof-node] Starting meshtastic worker...
[roof-node] ❌ Collection Failed: dial tcp 192.168.1.50:80: connect: connection refused

```

---

## 🏗 Building from Source

**Prerequisites:** Go 1.21+

1. **Clone the Repo:**

```bash
git clone [https://github.com/harborscale/lighthouse.git](https://github.com/harborscale/lighthouse.git)
cd lighthouse

```

2. **Build:**

```bash
# Linux / Mac
go build -o lighthouse cmd/lighthouse/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o lighthouse.exe cmd/lighthouse/main.go

```

3. **Cross-Compile (for Pi/Release):**

```bash
# Raspberry Pi (64-bit)
GOOS=linux GOARCH=arm64 go build -o lighthouse-pi cmd/lighthouse/main.go

```

---

## 🤖 Integration Drivers

Lighthouse currently supports the following drivers natively:

- **`linux` / `system**`: Uses `gopsutil` to fetch CPU, RAM, Disk, Uptime, and Load Avg.
- **`windows`**: Same as linux, but optimized for Windows WMI.
- **`meshtastic`**:
- **HTTP Mode:** If `ip` param is provided (`--param ip=1.2.3.4`), connects to device Wi-Fi JSON API.
- **Serial Mode:** If no IP, attempts to use Python CLI wrapper via USB.

---

### 🔌 Running Custom Scripts (The "Universal Adapter")

Lighthouse can turn **any** script into a telemetry stream. You don't need to import SDKs or manage HTTP requests in your script.

**How it works:**

1. Your script prints a JSON object to `STDOUT`.
2. Lighthouse captures it.
3. Lighthouse automatically tags it with your `ship_id` and the precise `timestamp`.
4. Lighthouse batches and transmits it to the cloud.

#### 1. The Requirement

Your script (Bash, Python, Node, Go, etc.) must output **valid JSON**.

**Example (`weather.py`):**

```python
import json
# Do your logic...
print(json.dumps({
    "temp_c": 24.5,
    "humidity": 60,
    "status": 1
}))

```

#### 2. The Command

Register the script using the `exec` source and pass the command string:

```bash
./lighthouse --add \
  --name "garden-pi" \
  --harbor-id "786" \
  --key "hs_live_key" \
  --source exec \
  --param command="python3 /home/user/scripts/weather.py"

```

#### 3. The Result

Lighthouse executes your command every interval (default 60s). It enriches the data before sending:

- **What your script output:** `{"temp_c": 24.5}`
- **What Harbor Scale receives:**

```json
{
  "ship_id": "garden-pi",
  "time": "2023-10-27T10:00:00.000Z",
  "cargo_id": "temp_c",
  "value": 24.5
}
```

#### 💡 Pro Tip: Bash One-Liners

You can even run simple inline shell commands without a file:

```bash
# Monitor the number of active SSH connections
./lighthouse --add \
  --name "ssh-monitor" \
  --harbor-id "786" \
  --source exec \
  --param command='echo "{\"active_ssh\": $(who | wc -l)}"'

```

---

## 🔄 Auto-Update Mechanism

Lighthouse checks GitHub Releases every 24 hours.

- If a new version (tag `v*`) is found, it downloads the binary for the current OS/Arch.
- It verifies the checksum.
- It replaces the running executable and restarts the service.
- **Disable this:** `./lighthouse --autoupdate=false`

---

## 📄 License

MIT License. Built with ❤️ for the Harbor Scale Community.
