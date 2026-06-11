package runner

import (
	"fmt"
	"sync"
	"time"

	"github.com/grimdork/climate/cfmt"
)

type TargetResult struct {
	Name     string
	Status   string
	Duration time.Duration
	Err      error
}

type TargetResults struct {
	mu  sync.Mutex
	res []TargetResult
}

func (tr *TargetResults) Add(name, status string, dur time.Duration, err error) {
	tr.mu.Lock()
	tr.res = append(tr.res, TargetResult{Name: name, Status: status, Duration: dur, Err: err})
	tr.mu.Unlock()
}

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

	fmt.Println("── Summary ──────────────────")
	fmt.Printf("%-*s  Duration   Result\n", maxName, "Target")
	for _, r := range tr.res {
		dur := r.Duration.Round(time.Millisecond).String()
		statusStr := r.Status
		switch r.Status {
		case "OK":
			statusStr = cfmt.Sprintf("%green %s%reset", r.Status)
		case "SKIPPED":
			statusStr = cfmt.Sprintf("%yellow %s%reset", r.Status)
		case "FAILED":
			statusStr = cfmt.Sprintf("%red %s%reset", r.Status)
		}
		fmt.Printf("%-*s  %-9s  %s\n", maxName, r.Name, dur, statusStr)
	}
	fmt.Println()
}

func cprintf(format string, args ...any) {
	fmt.Print(cfmt.Sprintf(format, args...))
}
