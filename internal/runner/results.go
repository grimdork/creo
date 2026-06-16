package runner

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/grimdork/climate/fx"
)

// TargetResult holds the outcome of a single target build.
type TargetResult struct {
	Name     string        // Target name from the fiat file
	Status   string        // "OK", "SKIPPED", or "FAILED"
	Duration time.Duration // Wall-clock time spent building
	Err      error         // Non-nil when Status is "FAILED"
}

// TargetResults collects per-target outcomes across a run.
type TargetResults struct {
	mu  sync.Mutex
	res []TargetResult
}

// Add appends a result in a thread-safe manner.
func (tr *TargetResults) Add(name, status string, dur time.Duration, err error) {
	tr.mu.Lock()
	tr.res = append(tr.res, TargetResult{Name: name, Status: status, Duration: dur, Err: err})
	tr.mu.Unlock()
}

// Print writes the summary table to stdout.
func (tr *TargetResults) Print() {
	if len(tr.res) == 0 {
		return
	}

	maxName := 6
	for _, r := range tr.res {
		if len(r.Name) > maxName {
			maxName = len(r.Name)
		}
	}

	var b strings.Builder
	b.WriteString("── Summary ──────────────────\n")
	fmt.Fprintf(&b, "%-*s  Duration   Result\n", maxName, "Target")
	for _, r := range tr.res {
		dur := r.Duration.Round(time.Millisecond).String()
		statusStr := r.Status
		switch r.Status {
		case "OK":
			statusStr = fx.Render("{success}{}{@}", r.Status)
		case "SKIPPED":
			statusStr = fx.Render("{warning}{}{@}", r.Status)
		case "FAILED":
			statusStr = fx.Render("{danger}{}{@}", r.Status)
		}
		fmt.Fprintf(&b, "%-*s  %-9s  %s\n", maxName, r.Name, dur, statusStr)
	}
	fx.Fprint(os.Stdout, b.String())
	fx.Println("")
}
