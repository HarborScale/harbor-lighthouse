

# 🚢 Harbor Lighthouse

**The Universal Telemetry Agent for [Harbor Scale**](https://harborscale.com)

Lighthouse is a tiny, single-binary agent that runs on any computer (Linux, Mac, Windows, Raspberry Pi). It collects data, handles network issues, updates itself automatically, and ships your metrics securely to the Harbor Scale cloud or your own [Self-Hosted OSS](https://github.com/harborscale/telemetry-harbor-oss) instance.

---

## ⚡ Installation

We provide a universal installer that automatically detects your OS and architecture.

### 🐧 Linux / 🍎 macOS / 🥧 Raspberry Pi

Copy and paste this into your terminal:

```bash
curl -sL get.harborscale.com | sudo bash

```

### 🪟 Windows (PowerShell)

Open PowerShell as Administrator and run:

```powershell
iwr get.harborscale.com | iex

```

> **Note:** This installs Lighthouse as a system service. It will start automatically on boot.

---

## 🎮 How to Use

Everything is done using the `lighthouse` command.

### 1. Add a Monitor (Cloud)

To start monitoring a device on **Harbor Scale Cloud**:

```bash
lighthouse --add \
  --name "server-01" \
  --harbor-id "123" \
  --key "hs_live_key_xxx" \
  --source linux

```

### 2. Add a Monitor (Self-Hosted / OSS) 🏠

To monitor a device on your own **[Harbor Scale OSS](https://github.com/harborscale/telemetry-harbor-oss)** instance:

```bash
lighthouse --add \
  --name "local-server" \
  --endpoint "http://192.168.1.50:8000" \
  --key "your_oss_api_key" \
  --source linux

```

> **Note:** When using `--endpoint`, the `--harbor-id` flag is optional.

### 3. Run a Custom Script

Turn any script into a telemetry stream.

```bash
lighthouse --add \
  --name "weather-script" \
  --harbor-id "123" \
  --key "hs_live_key_xxx" \
  --source exec \
  --param command="python3 /opt/weather.py"

```

### 4. Manage the Agent

| Command | Description |
| --- | --- |
| `sudo lighthouse --install` | **Start here.** Installs Lighthouse as a background service. |
| `lighthouse --list` | Shows the health status of all running monitors. |
| `lighthouse --logs "name"` | Shows the debug logs for a specific monitor. |
| `lighthouse --remove "name"` | Stops and deletes a monitor configuration. |
| `lighthouse --autoupdate=false` | Disables the automatic 24h update check. |
| `lighthouse --uninstall` | Removes the service and binary from your system. |

---

## ⚙️ Configuration Flags (Reference)

When running `lighthouse --add`, you can use these flags to customize behavior:

| Flag | Required? | Description | Default |
| --- | --- | --- | --- |
| `--name` | ✅ Yes | A unique ID for this device (e.g., `server-01`). | - |
| `--harbor-id` | ☁️ Cloud | Your Harbor ID (Required for Cloud). | - |
| `--endpoint` | 🏠 OSS | Custom API URL (Required for Self-Hosted). | `https://harborscale.com` |
| `--key` | ❌ No | Your API Key. | - |
| `--source` | ✅ Yes | Which collector to use (`linux`, `exec`, `uptime`, `docker`, `meshtastic`). | `linux` |
| `--interval` | ❌ No | How often to collect data (in seconds). | `60` |
| `--batch-size` | ❌ No | Max number of metrics to send in one HTTP request. | `100` |
| `--param` | ❌ No | Pass specific settings to a collector (e.g., `--param target_url=...`). | - |

---
## 🔌 Collectors & Examples

Lighthouse comes with built-in drivers called "Collectors". Choose one using `--source`.

### 1. System Monitors (`linux`, `windows`, `macos`)

Automatically collects CPU, RAM, Disk Usage, Uptime, and Load Averages.

**Example Command:**

```bash
lighthouse --add \
  --name "my-server" \
  --harbor-id "123" \
  --key "hs_live_key_xxx" \
  --source linux

```

* **Note:** On Windows or Mac, simply swap `--source linux` for `--source windows` or `--source macos`.

### 2. Docker Engine (`docker`)

Monitors the local Docker daemon. Captures container state (running/exited), uptime, and resource usage per container.

**Example Command:**

```bash
lighthouse --add \
  --name "docker-host" \
  --harbor-id "123" \
  --key "hs_live_key_xxx" \
  --source docker

```

* **Requirement:** The user running Lighthouse must have permission to access `/var/run/docker.sock` (usually the `docker` group).

### 3. HTTP Uptime (`uptime`)

Monitors website availability, response codes, and latency.

**Example Command:**

```bash
lighthouse --add \
  --name "website-monitor" \
  --harbor-id "123" \
  --key "hs_live_key_xxx" \
  --source uptime \
  --param target_url="https://google.com"

```

* **Optional Params:** `--param timeout_ms=5000` (Set connection timeout in milliseconds).

### 4. Custom Scripts (`exec`)

Runs **any** shell command or script (Python, Bash, Node, etc.). The script must output JSON to STDOUT.

**Example Command:**

```bash
lighthouse --add \
  --name "custom-script" \
  --harbor-id "123" \
  --key "hs_live_key_xxx" \
  --source exec \
  --param command="python3 /opt/sensor.py"

```

* **Optional Params:** `--param timeout_ms=10000` (Kill script if it hangs longer than this).

### 5. Meshtastic LoRa (`exec` + `mesh_engine`)

Ingests telemetry from a Meshtastic USB device. It acts as a gateway, reporting battery, environmental metrics, and signal stats for every node in your mesh.

**Step A: Install the Engine**

* **Linux / Mac / Pi:** `curl -sL get.harborscale.com/meshtastic | sudo bash`
* **Windows:** `iwr get.harborscale.com/meshtastic | iex`

**Step B: Add the Monitor**

```bash
lighthouse --add \
  --name "meshtastic-gateway" \
  --harbor-id "123" \
  --key "hs_live_key_xxx" \
  --source exec \
  --param command="mesh_engine --ttl 3600"

```

* **Optional Params:**
* `--ttl <seconds>`: Ignore nodes not heard from in X seconds (Default: 3600).
* `--port <port>`: Force specific USB serial port (e.g., `/dev/ttyUSB0` or `COM3`).


---

## 🛠️ Deep Dive: Custom Scripts (`exec`)

The `exec` collector allows you to integrate **any** data source. Lighthouse runs your command, captures STDOUT, and parses the JSON.

### Mode A: Single Ship (Simple)

Your script prints a single JSON object. Lighthouse assigns the `--name` you configured as the ID.

**Output:**

```json
{
  "temperature": 24.5,
  "humidity": 60
}

```

### Mode B: Many Ships (Advanced)

Your script acts as a gateway for multiple devices. It prints a JSON **Array** `[...]`. Lighthouse loops through the array and sends data for multiple ships at once.

**Output:**

```json
[
  {
    "ship_id": "sensor_living_room",
    "temperature": 22.0
  },
  {
    "ship_id": "sensor_kitchen",
    "temperature": 25.5
  }
]

```

> **Note:** If you provide `ship_id` in the JSON, it overrides the `--name` flag for that specific data point.

---

## 🏗 Building from Source

**Prerequisites:** Go 1.21+

1. **Clone the Repo:**

```bash
git clone https://github.com/harborscale/harbor-lighthouse.git
cd harbor-lighthouse

```

2. **Build:**

```bash
# Linux / Mac
go build -o lighthouse cmd/lighthouse/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o lighthouse.exe cmd/lighthouse/main.go

```

---

## 📄 License

MIT License. Built with ❤️ for the Harbor Scale Community.
