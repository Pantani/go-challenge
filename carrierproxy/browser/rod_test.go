package browser

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
)

// TestNewLauncher checks the launcher runs headless and uses the browser
// lookPath finds.
func TestNewLauncher(t *testing.T) {
	launcher := newLauncher(func() (string, bool) { return "/opt/chrome", true })
	if launcher.Get(flags.Bin) != "/opt/chrome" || !launcher.Has(flags.Headless) {
		t.Fatalf("launcher = %#v", launcher)
	}
}

// TestLaunchPageWithFailure points the launcher at a binary that exits
// immediately and checks the launch failure is reported without a closer.
func TestLaunchPageWithFailure(t *testing.T) {
	bin, err := exec.LookPath("true")
	if err != nil {
		t.Fatalf("find true: %v", err)
	}
	_, closePage, err := launchPageWith(launcher.New().Headless(true).Bin(bin), time.Second)
	if err == nil || !strings.Contains(err.Error(), "launch browser") {
		t.Fatalf("launchPageWith() error = %v", err)
	}
	if closePage != nil {
		t.Fatal("launchPageWith() returned a closer after failure")
	}
}
