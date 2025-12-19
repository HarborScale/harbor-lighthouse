package updater

import (
	"log"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

// CHANGE THIS TO YOUR REPO
const RepoSlug = "harborscale/harbor-lighthouse"

func StartBackgroundChecker(currentVer string, enabled bool) {
	if !enabled || currentVer == "dev" { return }

	// Check immediately once, then every 24 hours
	check(currentVer)

	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		check(currentVer)
	}
}

func check(currentVer string) {
	v, err := semver.Parse(currentVer)
	if err != nil { return } // invalid version string, skip

	latest, err := selfupdate.UpdateSelf(v, RepoSlug)
	if err != nil {
		log.Println("⚠️ Auto-Update Check Failed:", err)
		return
	}

	if !latest.Version.Equals(v) {
		log.Println("🚀 Updated to version", latest.Version, "- Restarting service...")
		// We rely on the service manager (systemd) to restart the process
		// if we exit. Or we can let the user restart manually.
		// For a service, exiting usually triggers a restart loop.
		panic("Restarting to apply update")
	}
}
