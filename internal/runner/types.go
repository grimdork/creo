package runner

import (
	"strings"
	"sync"

	"github.com/grimdork/climate/fx"
)

// CacheStats collects L1 (local) and L2 (remote) cache hit/miss counters
// for reporting. All writes are mutex-protected.
type CacheStats struct {
	mu       sync.Mutex
	L1Hits   int
	L1Misses int
	L2Hits   int
	L2Misses int
}

func (s *CacheStats) add(n *int) {
	s.mu.Lock()
	*n++
	s.mu.Unlock()
}

// L1Hit increments the local cache hit counter.
func (s *CacheStats) L1Hit() { s.add(&s.L1Hits) }

// L1Miss increments the local cache miss counter.
func (s *CacheStats) L1Miss() { s.add(&s.L1Misses) }

// L2Hit increments the remote (SSH) cache hit counter.
func (s *CacheStats) L2Hit() { s.add(&s.L2Hits) }

// L2Miss increments the remote (SSH) cache miss counter.
func (s *CacheStats) L2Miss() { s.add(&s.L2Misses) }

// Print writes cache statistics to stdout, protected by the mutex.
func (s *CacheStats) Print() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.L1Hits == 0 && s.L1Misses == 0 && s.L2Hits == 0 && s.L2Misses == 0 {
		return
	}
	totalL1 := s.L1Hits + s.L1Misses
	totalL2 := s.L2Hits + s.L2Misses
	fx.Println("{bold}Cache stats:{@}")
	if totalL1 > 0 {
		pct := float64(s.L1Hits) / float64(totalL1) * 100
		fx.Println("  L1: {} hits, {} misses ({:.0f}% hit rate)", s.L1Hits, s.L1Misses, pct)
	}
	if totalL2 > 0 {
		pct := float64(s.L2Hits) / float64(totalL2) * 100
		fx.Println("  L2: {} hits, {} misses ({:.0f}% hit rate)", s.L2Hits, s.L2Misses, pct)
	}
}

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
	CacheStats     *CacheStats
	Results        *TargetResults
}

// combo pairs an architecture, OS and binary path for iterating target permutations.
type combo struct {
	arch, osval, bin string
}
