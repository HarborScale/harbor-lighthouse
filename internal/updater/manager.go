package updater

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/rhysd/go-github-selfupdate/selfupdate"
)

// REPO_SLUG: The "Owner/Name" of the repository on GitHub.

const RepoSlug = "harborscale/harbor-lighthouse"

// StartBackgroundChecker runs strictly in the background.
// intended to be called via 'go updater.StartBackgroundChecker(...)'
func StartBackgroundChecker(currentVer string, enabled bool) {
	if !enabled || currentVer == "dev" {
		return
	}

	// Check immediately on startup
	check(currentVer)

	// Check every 24 hours
	ticker := time.NewTicker(24 * time.Hour)
	for range ticker.C {
		check(currentVer)
	}
}

func check(currentVer string) {
	// sanitize version (strip 'v' prefix if present)
	cleanVer := strings.TrimPrefix(currentVer, "v")
	v, err := semver.Parse(cleanVer)
	if err != nil {
		log.Printf("⚠️ Updater: Could not parse current version '%s': %v", currentVer, err)
		return
	}

	// UpdateSelf automatically:
	// 1. Checks GitHub Releases for a newer version
	// 2. Downloads the asset for the current OS/Arch
	// 3. Verifies checksums (if configured)
	// 4. Replaces the currently running binary
	latest, err := selfupdate.UpdateSelf(v, RepoSlug)
	if err != nil {
		// Network errors or no update found are common, just log and continue
		// Don't log "Update Failed" if it just meant "Already up to date"
		if !strings.Contains(err.Error(), "Current binary is the latest version") {
			log.Println("⚠️ Auto-Update Check Failed:", err)
		}
		return
	}

	if !latest.Version.Equals(v) {
		log.Printf("🚀 Updated to version %s (from %s)", latest.Version, v)
		log.Println("🔄 Exiting process to allow Service Manager to restart with new binary...")

		// Force exit with status 1.
		// - Systemd: With 'Restart=always' or 'Restart=on-failure', this triggers a restart.
		// - Windows: Service recovery settings usually restart on crash/exit.
		os.Exit(1)
	}
}
