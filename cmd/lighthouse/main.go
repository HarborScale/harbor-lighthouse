package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/harborscale/harbor-lighthouse/internal/collectors"
	"github.com/harborscale/harbor-lighthouse/internal/config"
	"github.com/harborscale/harbor-lighthouse/internal/engine"
	"github.com/harborscale/harbor-lighthouse/internal/service"
	"github.com/harborscale/harbor-lighthouse/internal/status"
	"github.com/harborscale/harbor-lighthouse/internal/transport"
	"github.com/harborscale/harbor-lighthouse/internal/updater"
)

//go:embed definitions.json
var definitionsBytes []byte

// Set by GitHub Actions build flags
var Version = "dev"

// --- Helper for Map Flags ---
type paramFlags map[string]string

func (i *paramFlags) String() string { return "params" }
func (i *paramFlags) Set(value string) error {
	p := strings.SplitN(value, "=", 2)
	if len(p) == 2 {
		(*i)[p[0]] = p[1]
	}
	return nil
}

func main() {
	// System Flags
	install := flag.Bool("install", false, "Install as Service")
	uninstall := flag.Bool("uninstall", false, "Uninstall Service")
	list := flag.Bool("list", false, "Show Status")
	logs := flag.String("logs", "", "Show logs for instance")
	ver := flag.Bool("version", false, "Show version")

	// Config Flags
	add := flag.Bool("add", false, "Add Instance")
	remove := flag.String("remove", "", "Remove Instance")
	autoUpdate := flag.String("autoupdate", "", "true/false")

	// Instance Params
	name := flag.String("name", "", "Instance Name (ship_id)")
	harborID := flag.String("harbor-id", "", "Harbor ID")
	key := flag.String("key", "", "API Key")
	src := flag.String("source", "linux", "Source (linux, exec)")
	typ := flag.String("type", "general", "Harbor Type (general, gps)")

	// OSS / Custom Endpoint Flag
	endpoint := flag.String("endpoint", "", "Custom API URL (e.g. http://localhost:8080)")

	// Rate Limiting Flags
	interval := flag.Int("interval", 60, "Collection interval in seconds")
	batchSize := flag.Int("batch-size", 100, "Max items per HTTP request")

	params := make(paramFlags)
	flag.Var(&params, "param", "Key=Value params")

	flag.Parse()

	// 0. Version
	if *ver {
		fmt.Printf("Harbor Lighthouse %s\n", Version)
		return
	}

	// 1. Load Brain
	if err := engine.Load(definitionsBytes); err != nil {
		log.Fatalf("❌ Fatal: Definitions corrupted: %v", err)
	}

	// 2. Global Settings
	if *autoUpdate != "" {
		cfg, _ := config.Load()
		cfg.AutoUpdate = (*autoUpdate == "true")
		config.Save(cfg)
		fmt.Printf("✅ Auto-Update set to: %v\n", cfg.AutoUpdate)
		return
	}

	// 3. Service Control
	svc, err := service.Setup(runDaemon)
	if err != nil {
		log.Fatal("Service Setup Error:", err)
	}

	if *install {
		fmt.Println("Installing Service...")
		if err := svc.Install(); err != nil {
			log.Fatal("❌ Install Failed:", err)
		}
		if err := svc.Start(); err != nil {
			log.Fatal("❌ Start Failed:", err)
		}
		fmt.Println("✅ Service Installed & Started!")
		return
	}
	if *uninstall {
		svc.Stop()
		svc.Uninstall()
		fmt.Println("🗑️ Service Removed.")
		return
	}

	// 4. CLI Commands (List/Logs/Add/Remove)
	if *list {
		showStatus()
		return
	}
	if *logs != "" {
		showLogsFor(*logs)
		return
	}

	if *add {
		// VALIDATION LOGIC
		if *name == "" {
			log.Fatal("❌ Error: --name is required")
		}
		// If using Cloud (no endpoint), HarborID is required.
		// If using OSS (endpoint set), HarborID is optional.
		if *endpoint == "" && *harborID == "" {
			log.Fatal("❌ Error: --harbor-id is required for Cloud usage")
		}

		cfg, _ := config.Load()
		instance := config.Instance{
			Name: *name, HarborID: *harborID, APIKey: *key,
			Source: *src, HarborType: *typ, Params: params,
			Interval:     *interval,
			MaxBatchSize: *batchSize,
			Endpoint:     *endpoint, // Save custom URL
		}
		if err := cfg.Add(instance); err != nil {
			log.Fatal("❌", err)
		}
		config.Save(cfg)
		fmt.Println("✅ Added instance.")
		reloadService() // Auto Restart
		return
	}

	if *remove != "" {
		cfg, _ := config.Load()
		if cfg.Remove(*remove) {
			config.Save(cfg)
			fmt.Println("🗑️ Removed instance.")
			reloadService() // Auto Restart
		} else {
			fmt.Println("❌ Instance not found.")
		}
		return
	}

	// 5. Run Daemon (Foreground or via Service)
	if err := svc.Run(); err != nil {
		log.Fatal(err)
	}
}

func runDaemon() {
	setupLogging()
	log.Printf("🚢 Harbor Lighthouse %s Starting...", Version)

	cfg, _ := config.Load()

	// Background Auto-Update
	go updater.StartBackgroundChecker(Version, cfg.AutoUpdate)

	// --- IDLE MODE (Prevents Service Crash) ---
	if len(cfg.Instances) == 0 {
		log.Println("💤 No instances configured. Entering Idle Mode.")
		for {
			time.Sleep(1 * time.Hour)
		}
	}

	// --- WORKER MODE ---
	var wg sync.WaitGroup
	for _, inst := range cfg.Instances {
		wg.Add(1)
		go func(i config.Instance) {
			defer wg.Done()
			worker(i)
		}(inst)
	}
	wg.Wait()
}

func worker(inst config.Instance) {
	prefix := fmt.Sprintf("[%s]", inst.Name)

	if inst.Interval < 1 { inst.Interval = 60 }
	if inst.MaxBatchSize < 1 { inst.MaxBatchSize = 100 }

	// 1. Get Mode & Suffix
	def, err := engine.Get(inst.HarborType)
	if err != nil {
		log.Printf("%s ❌ Configuration Error: Unknown Type '%s'", prefix, inst.HarborType)
		status.Update(inst.Name, err)
		return
	}

	// --- 🔗 INTELLIGENT URL BUILDER ---
	var url string
	if inst.Endpoint != "" {
		// OSS MODE: Use custom endpoint, NO Harbor ID in path
		// Format: {Endpoint}/api/v2/ingest{Suffix}
		cleanBase := strings.TrimRight(inst.Endpoint, "/")
		url = fmt.Sprintf("%s/api/v2/ingest%s", cleanBase, def.EndpointSuffix)
	} else {
		// CLOUD MODE: Use standard URL, INCLUDE Harbor ID
		// Format: https://harborscale.com/api/v2/ingest/{ID}{Suffix}
		url = fmt.Sprintf("https://harborscale.com/api/v2/ingest/%s%s", inst.HarborID, def.EndpointSuffix)
	}

	// 2. Get Collector
	col, err := collectors.Get(inst.Source)
	if err != nil {
		log.Printf("%s ❌ Collector Error: %v", prefix, err)
		status.Update(inst.Name, err)
		return
	}

	log.Printf("%s Started (%s mode) -> %s", prefix, def.Mode, url)

	ticker := time.NewTicker(time.Duration(inst.Interval) * time.Second)
	defer ticker.Stop()

	// 3. Collection Loop
	for {
		<-ticker.C

		data, err := col(inst.Params)
		if err != nil {
			log.Printf("%s ❌ Collection Failed: %v", prefix, err)
			status.Update(inst.Name, err)
			continue
		}

		if def.Mode == "cargo" {
			// --- CARGO MODE (BATCHED) ---
			var fullList []transport.CargoPayload

			for k, v := range data {
				fullList = append(fullList, transport.CargoPayload{
					Time:    time.Now().UTC().Format(time.RFC3339Nano),
					ShipID:  inst.Name,
					CargoID: k,
					Value:   v,
				})
			}

			// Chunking logic
			totalItems := len(fullList)
			for i := 0; i < totalItems; i += inst.MaxBatchSize {
				end := i + inst.MaxBatchSize
				if end > totalItems { end = totalItems }

				batchChunk := fullList[i:end]
				err := transport.SendBatch(url+"/batch", inst.APIKey, batchChunk)

				if err != nil {
					log.Printf("%s ⚠️ Batch Send Error: %v", prefix, err)
					if strings.Contains(err.Error(), "429") {
						log.Printf("%s 🛑 Rate Limit Hit! Cooling down 60s...", prefix)
						time.Sleep(60 * time.Second)
					}
					status.Update(inst.Name, err)
				} else {
					status.Update(inst.Name, nil)
				}
			}

		} else {
			// --- RAW MODE (GPS/Single) ---
			data["time"] = time.Now().UTC().Format(time.RFC3339Nano)
			data["ship_id"] = inst.Name

			if err := transport.Send(url, inst.APIKey, data); err != nil {
				log.Printf("%s ⚠️ Send Fail: %v", prefix, err)
				status.Update(inst.Name, err)
			} else {
				status.Update(inst.Name, nil)
			}
		}
	}
}

// --- HELPER FUNCTIONS ---

func setupLogging() {
	f, err := os.OpenFile("lighthouse.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stdout, f))
}

func showStatus() {
	cfg, _ := config.Load()
	st := status.Load()
	fmt.Printf("📋 Harbor Lighthouse %s\n", Version)
	if len(cfg.Instances) == 0 {
		fmt.Println("   (No instances)")
	}

	for _, i := range cfg.Instances {
		s := st[i.Name]
		icon := "🔴"
		msg := "Offline"
		if s.LastContact > 0 {
			ago := time.Since(time.Unix(s.LastContact, 0)).Round(time.Second)
			if s.Healthy {
				icon = "🟢"
				msg = fmt.Sprintf("Healthy (%s ago)", ago)
			} else {
				icon = "⚠️ "
				msg = fmt.Sprintf("Error: %s (%s ago)", s.LastError, ago)
			}
		}
		// Clean display logic for missing HarborID in OSS mode
		hID := i.HarborID
		if hID == "" {
			hID = "OSS"
		}
		fmt.Printf("%s [%s] %s -> %s\n     └─ %s\n", icon, i.Name, i.Source, hID, msg)
	}
}

func showLogsFor(n string) {
	d, _ := os.ReadFile("lighthouse.log")
	lines := strings.Split(string(d), "\n")
	t := fmt.Sprintf("[%s]", n)
	for _, l := range lines {
		if strings.Contains(l, t) {
			fmt.Println(l)
		}
	}
}

// reloadService attempts to restart the background service to apply config changes.
// reloadService attempts to restart the background service to apply config changes.
func reloadService() {
	fmt.Println("⚙️  Applying changes...")

	var cmd *exec.Cmd
	var restartCmd string

	if runtime.GOOS == "windows" {
		// Windows: Requires Admin privileges to run this
		cmd = exec.Command("powershell", "-Command", "Restart-Service", "HarborLighthouse")
		restartCmd = "Restart-Service HarborLighthouse (Run as Admin)"
	} else {
		// Linux/Mac
		cmd = exec.Command("sudo", "systemctl", "restart", "HarborLighthouse")
		restartCmd = "sudo lighthouse --install"
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Could not auto-restart service: %v\n", err)
		
		if runtime.GOOS == "windows" {
			fmt.Println("👉 NOTE: On Windows, you must run your terminal as Administrator to restart the service automatically.")
			fmt.Println("👉 Manual Fix: Open PowerShell as Admin and run: Restart-Service HarborLighthouse")
		} else {
			fmt.Printf("👉 Manual Fix: Run '%s'\n", restartCmd)
		}
	} else {
		fmt.Println("♻️  Service restarted successfully!")
	}
}
