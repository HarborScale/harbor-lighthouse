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

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/harborscale/harbor-lighthouse/internal/collectors"
	"github.com/harborscale/harbor-lighthouse/internal/config"
	"github.com/harborscale/harbor-lighthouse/internal/engine"
	"github.com/harborscale/harbor-lighthouse/internal/service"
	"github.com/harborscale/harbor-lighthouse/internal/status"
	"github.com/harborscale/harbor-lighthouse/internal/transport"
	"github.com/harborscale/harbor-lighthouse/internal/updater"
)

const (
	BinaryName  = "lighthouse"
	ServiceName = "harbor-lighthouse"
	AppDisplay  = "Harbor Lighthouse"
)

//go:embed definitions.json
var definitionsBytes []byte

var Version = "dev"

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
	// 1. INITIALIZE CONFIG
	// This sets up config.GlobalDir and config.GlobalLogPath
	config.Initialize()

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

	endpoint := flag.String("endpoint", "", "Custom API URL")
	interval := flag.Int("interval", 60, "Collection interval")
	batchSize := flag.Int("batch-size", 100, "Max items per request")

	params := make(paramFlags)
	flag.Var(&params, "param", "Key=Value params")

	flag.Parse()

	// 0. Version
	if *ver {
		fmt.Printf("%s v%s\n", AppDisplay, Version)
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
		
		if err := config.Save(cfg); err != nil {
			log.Fatalf("❌ Failed to save config: %v (Try running with sudo)", err)
		}
		
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
		fmt.Printf("   Binary: %s\n", os.Args[0])
		// Use the centralized config variable
		fmt.Printf("   Config: %s\n", config.GlobalDir)

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

	// 4. CLI Commands
	if *list {
		showStatus()
		return
	}
	if *logs != "" {
		showLogsFor(*logs)
		return
	}

	if *add {
		if *name == "" {
			log.Fatal("❌ Error: --name is required")
		}
		if *endpoint == "" && *harborID == "" {
			log.Fatal("❌ Error: --harbor-id is required for Cloud usage")
		}

		cfg, _ := config.Load()
		instance := config.Instance{
			Name: *name, HarborID: *harborID, APIKey: *key,
			Source: *src, HarborType: *typ, Params: params,
			Interval:     *interval,
			MaxBatchSize: *batchSize,
			Endpoint:     *endpoint,
		}
		if err := cfg.Add(instance); err != nil {
			log.Fatal("❌", err)
		}
		
		if err := config.Save(cfg); err != nil {
			log.Fatalf("❌ Failed to save config: %v (Try running with sudo)", err)
		}
		
		config.Save(cfg)
		fmt.Println("✅ Added instance.")
		reloadService()
		return
	}

	if *remove != "" {
		cfg, _ := config.Load()
		if cfg.Remove(*remove) {
			
			if err := config.Save(cfg); err != nil {
				log.Fatalf("❌ Failed to save config: %v (Try running with sudo)", err)
			}
			
			fmt.Println("🗑️ Removed instance.")
			reloadService()
		} else {
			fmt.Println("❌ Instance not found.")
		}
		return
	}

	// 5. Run Daemon
	if err := svc.Run(); err != nil {
		log.Fatal(err)
	}
}

func runDaemon() {
	setupLogging()
	log.Printf("🚢 %s %s Starting...", AppDisplay, Version)
	// Use centralized config variable
	log.Printf("📂 Config Dir: %s", config.GlobalDir)

	cfg, _ := config.Load()

	// Background Auto-Update
	go updater.StartBackgroundChecker(Version, cfg.AutoUpdate)

	// --- IDLE MODE ---
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

	if inst.Interval < 1 {
		inst.Interval = 60
	}
	if inst.MaxBatchSize < 1 {
		inst.MaxBatchSize = 100
	}

	def, err := engine.Get(inst.HarborType)
	if err != nil {
		log.Printf("%s ❌ Configuration Error: Unknown Type '%s'", prefix, inst.HarborType)
		status.Update(inst.Name, err)
		return
	}

	var url string
	if inst.Endpoint != "" {
		cleanBase := strings.TrimRight(inst.Endpoint, "/")
		url = fmt.Sprintf("%s/api/v2/ingest%s", cleanBase, def.EndpointSuffix)
	} else {
		url = fmt.Sprintf("https://harborscale.com/api/v2/ingest/%s%s", inst.HarborID, def.EndpointSuffix)
	}

	col, err := collectors.Get(inst.Source)
	if err != nil {
		log.Printf("%s ❌ Collector Error: %v", prefix, err)
		status.Update(inst.Name, err)
		return
	}

	log.Printf("%s Started (%s mode) -> %s", prefix, def.Mode, url)

	ticker := time.NewTicker(time.Duration(inst.Interval) * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C

		shipResults, err := col(inst.Params)
		if err != nil {
			log.Printf("%s ❌ Collection Failed: %v", prefix, err)
			status.Update(inst.Name, err)
			continue
		}

		currentTime := time.Now().UTC().Format(time.RFC3339Nano)

		if def.Mode == "cargo" {
			var batchBuffer []transport.CargoPayload

			for _, data := range shipResults {
				activeShipID := inst.Name
				if sid, ok := data["ship_id"].(string); ok && sid != "" {
					activeShipID = sid
					delete(data, "ship_id")
				}

				for k, v := range data {
					batchBuffer = append(batchBuffer, transport.CargoPayload{
						Time:    currentTime,
						ShipID:  activeShipID,
						CargoID: k,
						Value:   v,
					})
				}
			}

			totalItems := len(batchBuffer)
			if totalItems > 0 {
				for i := 0; i < totalItems; i += inst.MaxBatchSize {
					end := i + inst.MaxBatchSize
					if end > totalItems {
						end = totalItems
					}

					chunk := batchBuffer[i:end]
					if err := transport.SendBatch(url+"/batch", inst.APIKey, chunk); err != nil {
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
			}

		} else {
			for _, data := range shipResults {
				activeShipID := inst.Name
				if sid, ok := data["ship_id"].(string); ok && sid != "" {
					activeShipID = sid
					data["ship_id"] = sid
				} else {
					data["ship_id"] = activeShipID
				}

				data["time"] = currentTime

				if err := transport.Send(url, inst.APIKey, data); err != nil {
					log.Printf("%s ⚠️ Send Fail (%s): %v", prefix, activeShipID, err)
					status.Update(inst.Name, err)
				} else {
					status.Update(inst.Name, nil)
				}
			}
		}
	}
}

// bestEffort wraps a writer so a failing destination cannot silence the
// others in an io.MultiWriter. MultiWriter aborts on the first writer that
// returns an error, so with a bare (os.Stdout, file) pair a Windows service
// -- which has no console, and whose stdout handle is therefore invalid --
// would drop every line before it ever reached the log file. Reporting a
// full successful write keeps the remaining writers in the chain running.
type bestEffort struct{ w io.Writer }

func (b bestEffort) Write(p []byte) (int, error) {
	_, _ = b.w.Write(p)
	return len(p), nil
}

func setupLogging() {
	// Rotate the log so a long-running service cannot fill the disk.
	// The previous implementation opened the file with O_APPEND and never
	// truncated it, so on any persistent error (unreachable harbor, bad API
	// key, rate limiting) the per-cycle error lines grew without bound.
	// Caps usage at MaxSize + MaxBackups compressed files.
	rotating := &lumberjack.Logger{
		Filename:   config.GlobalLogPath,
		MaxSize:    10, // megabytes per file before rotating
		MaxBackups: 3,  // retained rotated files
		MaxAge:     28, // days
		Compress:   true,
	}

	log.SetOutput(io.MultiWriter(bestEffort{os.Stdout}, bestEffort{rotating}))
}

func showStatus() {
	cfg, _ := config.Load()
	st := status.Load()
	fmt.Printf("📋 %s %s\n", AppDisplay, Version)
	// Use centralized config variable
	fmt.Printf("📂 Config: %s\n", config.GlobalDir)

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
		hID := i.HarborID
		if hID == "" {
			hID = "OSS"
		}
		fmt.Printf("%s [%s] %s -> %s\n     └─ %s\n", icon, i.Name, i.Source, hID, msg)
	}
}

func showLogsFor(n string) {
	// Use centralized config variable
	d, err := os.ReadFile(config.GlobalLogPath)
	if err != nil {
		fmt.Printf("❌ Could not read logs at %s: %v\n", config.GlobalLogPath, err)
		return
	}
	lines := strings.Split(string(d), "\n")
	t := fmt.Sprintf("[%s]", n)
	found := false
	for _, l := range lines {
		if strings.Contains(l, t) {
			fmt.Println(l)
			found = true
		}
	}
	if !found {
		fmt.Println("No logs found for this instance in current log file.")
	}
}

func reloadService() {
	fmt.Println("⚙️  Applying changes...")

	var cmd *exec.Cmd
	var restartCmd string

	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-Command", "Restart-Service", ServiceName)
		restartCmd = fmt.Sprintf("Restart-Service %s (Run as Admin)", ServiceName)
	} else {
		cmd = exec.Command("sudo", "systemctl", "restart", ServiceName)
		restartCmd = fmt.Sprintf("sudo %s --install", BinaryName)
	}

	if err := cmd.Run(); err != nil {
		fmt.Printf("⚠️  Could not auto-restart service: %v\n", err)
		if runtime.GOOS == "windows" {
			fmt.Println("👉 NOTE: Run terminal as Administrator to restart automatically.")
		}
		fmt.Printf("👉 Manual Fix: Run '%s'\n", restartCmd)
	} else {
		fmt.Println("♻️  Service restarted successfully!")
	}
}
