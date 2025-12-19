# 🚢 Harbor Lighthouse

**The Universal Telemetry Agent for [Harbor Scale](https://harborscale.com)**

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

> **Note:** When using `--endpoint`, the `--harbor-id` flag is optional (as OSS is single-tenant).

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
| `--source` | ✅ Yes | Which collector to use (`linux`, `windows`, `exec`). | `linux` |
| `--interval` | ❌ No | How often to collect data (in seconds). | `60` |
| `--batch-size` | ❌ No | Max number of metrics to send in one HTTP request. | `10` |
| `--param` | ❌ No | Pass specific settings to a collector. | - |

---

## 🔌 Collectors

Lighthouse comes with built-in drivers called "Collectors". Choose one using `--source`.

### 1. System Monitors (`linux`, `windows`, `macos`)

Automatically collects CPU, RAM, Disk Usage, Uptime, and Load Averages.

* **Usage:** `--source linux`

### 2. Custom Scripts (`exec`)

Runs **any** shell command or script you write. The script must output **JSON**.

* **Usage:** `--source exec --param command="bash /home/user/script.sh"`

---

## 🛠️ Deep Dive: Custom Scripts (`exec`)

The `exec` collector allows you to integrate **any** data source (Python, Node, Bash, Go) without importing SDKs.

### How it works

1. Your script prints a JSON object to `STDOUT`.
2. Lighthouse captures it.
3. Lighthouse automatically tags it with your `ship_id` and timestamp.
4. Lighthouse "explodes" the JSON keys into separate metrics and batches them.

**✅ Correct Output:**

```json
{
  "temperature": 24.5,
  "humidity": 60,
  "voltage": 5.2
}

```

**❌ Incorrect Output (Do NOT use Arrays):**

```json
[
  {"temperature": 24.5},
  {"humidity": 60}
]

```

---

## 🏗 Building from Source

**Prerequisites:** Go 1.21+

1. **Clone the Repo:**
```bash
git clone [https://github.com/harborscale/harbor-lighthouse.git](https://github.com/harborscale/harbor-lighthouse.git)
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
