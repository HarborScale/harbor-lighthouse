
# 🚢 Harbor Lighthouse

**The Universal Telemetry Agent for [Harbor Scale](https://harborscale.com)**

Lighthouse is a tiny, single-binary agent that runs on any computer (Linux, Mac, Windows, Raspberry Pi). It collects data, handles network issues, updates itself automatically, and ships your metrics securely to the Harbor Scale cloud.



---
## ⚡ Installation

We provide a universal installer that automatically detects your OS and architecture.

### 🐧 Linux / 🍎 macOS / 🥧 Raspberry Pi
Copy and paste this into your terminal:
```bash
curl -sL get.harborscale.com | sudo bash
```
🪟 Windows (PowerShell)

Open PowerShell as Administrator and run:

```PowerShell
iwr get.harborscale.com | iex
```
> **Note:** This installs Lighthouse as a system service. It will start automatically on boot.
---

## 🎮 How to Use

Everything is done using the `./lighthouse` command.

### 1. Add a Monitor

To start monitoring a device, you "add" an instance.

**Example: Monitor this Linux Server**

```bash
lighthouse --add \
  --name "server-01" \
  --harbor-id "786" \
  --key "hs_live_key_xxx" \
  --source linux

```

### 2. Run a Custom Script

Turn any script into a telemetry stream.

```bash
lighthouse --add \
  --name "weather-script" \
  --harbor-id "786" \
  --key "hs_live_key_xxx" \
  --source exec \
  --param command="python3 /opt/weather.py"

```

### 3. Manage the Agent

| Command | Description |
| --- | --- |
| `sudo ./lighthouse --install` | **Start here.** Installs Lighthouse as a background service (Systemd/Launchd). |
| `./lighthouse --list` | Shows the health status of all running monitors. |
| `./lighthouse --logs "name"` | Shows the debug logs for a specific monitor. |
| `./lighthouse --remove "name"` | Stops and deletes a monitor configuration. |
| `./lighthouse --autoupdate=false` | Disables the automatic 24h update check. |

---

## ⚙️ Configuration Flags (Reference)

When running `lighthouse --add`, you can use these flags to customize behavior:

| Flag | Required? | Description | Default |
| --- | --- | --- | --- |
| `--name` | ✅ Yes | A unique ID for this device (e.g., `server-01`). | - |
| `--harbor-id` | ✅ Yes | Your Harbor ID from the dashboard. | - |
| `--key` | ❌ No | Your API Key (if required by your Harbor). | - |
| `--source` | ✅ Yes | Which collector to use (`linux`, `windows`, `exec`). | `linux` |
| `--interval` | ❌ No | How often to collect data (in seconds). | `60` |
| `--batch-size` | ❌ No | Max number of metrics to send in one HTTP request. | `10` |
| `--param` | ❌ No | Pass specific settings to a collector. | - |

---

## 🔌 Collectors

Lighthouse comes with built-in drivers called "Collectors". Choose one using `--source`.

### 1. System Monitors (`linux`, `windows`, `macos`)

Automatically collects CPU, RAM, Disk Usage, Uptime, and Load Averages.

* **Parameters:** None.
* **Usage:** `--source linux`

### 2. Custom Scripts (`exec`)

Runs **any** shell command or script you write. The script must output **JSON**.

* **Parameters:**
* `command` (Required): The full command string to run.


* **Usage:** `--source exec --param command="bash /home/user/script.sh"`

---

## 🛠️ Deep Dive: Custom Scripts (`exec`)

The `exec` collector allows you to integrate **any** data source (Python, Node, Bash, Go) without importing SDKs.

### How it works

1. Your script prints a JSON object to `STDOUT`.
2. Lighthouse captures it.
3. Lighthouse automatically tags it with your `ship_id` and the precise `timestamp`.
4. Lighthouse "explodes" the JSON keys into separate metrics and batches them to the cloud.

### ⚡ Handling Multiple Metrics

If your script collects multiple data points at once (e.g., temperature AND humidity), simply include them all as keys in **one single JSON object**.

**✅ Correct Output:**

```json
{
  "temperature": 24.5,
  "humidity": 60,
  "voltage": 5.2
}

```

*Lighthouse will split this into 3 separate metrics automatically.*

**❌ Incorrect Output (Do NOT use Arrays):**

```json
[
  {"temperature": 24.5},
  {"humidity": 60}
]

```

### Example: Multi-Metric Python Script

Save this as `sensors.py`:

```python
import json
import random

# Collect your data...
cpu_temp = 45.2
fan_speed = 1200
errors = 0

# Print ONE object with all data
print(json.dumps({
    "cpu_temp_c": cpu_temp,
    "fan_rpm": fan_speed,
    "error_count": errors
}))

```

**Run it:**

```bash
./lighthouse --add \
  --name "system-sensors" \
  --harbor-id "786" \
  --source exec \
  --param command="python3 sensors.py"

```

---

## 🧑‍💻 Contributing: How to Add New Collectors

Want to add a new integration (e.g., `docker`, `mysql`)? It's easy!

1. **Create the File:**
Create a new file in `internal/collectors/my_collector.go`.
2. **Write the Logic:**
Your function must match this signature:
```go
func MyCollector(params map[string]string) (map[string]interface{}, error) {
    // 1. Read params (e.g. params["port"])
    // 2. Collect data
    // 3. Return map[string]interface{}{"metric_name": 123}
    return data, nil
}

```


3. **Register It:**
Edit `internal/collectors/registry.go` and add your name to the switch statement:
```go
case "my_collector":
    return MyCollector, nil

```


4. **Build:** Run `go build`. You can now use `--source my_collector`.

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



---

## 📄 License

MIT License. Built with ❤️ for the Harbor Scale Community.

