package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
	_ "embed"

	"github.com/harborscale/lighthouse/internal/collectors"
	"github.com/harborscale/lighthouse/internal/config"
	"github.com/harborscale/lighthouse/internal/engine"
	"github.com/harborscale/lighthouse/internal/service"
	"github.com/harborscale/lighthouse/internal/status"
	"github.com/harborscale/lighthouse/internal/transport"
	"github.com/harborscale/lighthouse/internal/updater"
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
	if len(p) == 2 { (*i)[p[0]] = p[1] }
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
	src := flag.String("source", "linux", "Source (linux, meshtastic)")
	typ := flag.String("type", "general", "Harbor Type (general, gps)")

	var params = make(paramFlags)
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
	if err != nil { log.Fatal("Service Setup Error:", err) }

	if *install {
		fmt.Println("Installing Service...")
		if err := svc.Install(); err != nil { log.Fatal("❌ Install Failed:", err) }
		if err := svc.Start(); err != nil { log.Fatal("❌ Start Failed:", err) }
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
	if *list { showStatus(); return }
	if *logs != "" { showLogsFor(*logs); return }

	if *add {
		if *name == "" || *harborID == "" { log.Fatal("❌ Error: --name and --harbor-id are required") }
		cfg, _ := config.Load()
		instance := config.Instance{
			Name: *name, HarborID: *harborID, APIKey: *key,
			Source: *src, HarborType: *typ, Params: params,
		}
		if err := cfg.Add(instance); err != nil { log.Fatal("❌", err) }
		config.Save(cfg)
		fmt.Println("✅ Added instance. Restart service to apply.")
		return
	}

	if *remove != "" {
		cfg, _ := config.Load()
		if cfg.Remove(*remove) {
			config.Save(cfg)
			fmt.Println("🗑️ Removed instance.")
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

	if len(cfg.Instances) == 0 {
		log.Println("⚠️  No instances configured. Sleeping.")
	}

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

	// 1. Get Mode & URL
	def, err := engine.Get(inst.HarborType)
	if err != nil {
		log.Printf("%s ❌ Configuration Error: Unknown Type '%s'", prefix, inst.HarborType)
		status.Update(inst.Name, err)
		return
	}

	url := fmt.Sprintf("https://harborscale.com/api/v2/ingest/%s%s", inst.HarborID, def.EndpointSuffix)

	// 2. Get Collector
	col, err := collectors.Get(inst.Source)
	if err != nil {
		log.Printf("%s ❌ Collector Error: %v", prefix, err)
		status.Update(inst.Name, err)
		return
	}

	log.Printf("%s Started (%s mode) -> %s", prefix, def.Mode, url)

	// 3. Loop
	for {
		data, err := col(inst.Params)
		if err != nil {
			log.Printf("%s ⚠️ Collection Failed: %v", prefix, err)
			status.Update(inst.Name, err)
		} else {
			// Success Gathering

			if def.Mode == "cargo" {
				// --- CARGO MODE (Fan-Out) ---
				errCount := 0
				for k, v := range data {
					payload := transport.CargoPayload{
						Time: time.Now().UTC().Format(time.RFC3339Nano),
						ShipID: inst.Name,
						CargoID: k,
						Value: v,
					}
					if err := transport.Send(url, inst.APIKey, payload); err != nil {
						log.Printf("%s ⚠️ Send Fail (%s): %v", prefix, k, err)
						errCount++
					}
				}
				if errCount == 0 { status.Update(inst.Name, nil) }

			} else {
				// --- RAW MODE (Batch/GPS) ---
				// Inject meta
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

		// Wait 60s (could be configurable via params)
		time.Sleep(60 * time.Second)
	}
}

func setupLogging() {
	f, err := os.OpenFile("lighthouse.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil { return }
	log.SetOutput(io.MultiWriter(os.Stdout, f))
}

func showStatus() {
	cfg, _ := config.Load()
	st := status.Load()
	fmt.Printf("📋 Harbor Lighthouse %s\n", Version)
	if len(cfg.Instances) == 0 { fmt.Println("   (No instances)") }

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
		fmt.Printf("%s [%s] %s -> %s\n     └─ %s\n", icon, i.Name, i.Source, i.HarborID, msg)
	}
}

func showLogsFor(n string) {
	d, _ := os.ReadFile("lighthouse.log")
	lines := strings.Split(string(d), "\n")
	t := fmt.Sprintf("[%s]", n)
	for _, l := range lines {
		if strings.Contains(l, t) { fmt.Println(l) }
	}
}
