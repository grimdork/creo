package runner

import (
	"strings"
	"sync"
)

// Outputs stores per-target build output paths, keyed by target/arch+os, with mutex-protected access.
type Outputs struct {
	mu sync.RWMutex
	m  map[string]string
}

// Store saves a binary path for a target/arch/os combination.
func (o *Outputs) Store(target, arch, os, bin string) {
	o.mu.Lock()
	o.m[target+"/"+arch+"+"+os] = bin
	o.mu.Unlock()
}

// Load retrieves a previously stored binary path for a target/arch/os combination.
func (o *Outputs) Load(target, arch, os string) string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.m[target+"/"+arch+"+"+os]
}

// LoadAll returns all arch+os keys stored for a given target.
func (o *Outputs) LoadAll(target string) []string {
	o.mu.RLock()
	defer o.mu.RUnlock()
	prefix := target + "/"
	var out []string
	for k := range o.m {
		if strings.HasPrefix(k, prefix) {
			out = append(out, k[len(prefix):])
		}
	}
	return out
}

// RunOpts holds options that control the build/run lifecycle across all targets.
type RunOpts struct {
	Rebuild        bool
	Clean          bool
	Recursive      bool
	Verbose        bool
	Jobs           int
	KeepGoing      bool
	DryRun         bool
	RefreshCACerts bool
	BuildDir       string
	NoColor        bool
	CacheRemote    string
	Results        *TargetResults
}

// combo pairs an architecture, OS and binary path for iterating target permutations.
type combo struct {
	arch, osval, bin string
}
