package service

import (
	"strings"
	"sync"
	"time"
)

const (
	openAIModelLatencyCircuitFailureThreshold = 2
	openAIModelLatencyCircuitWindow           = time.Minute
	openAIModelLatencyCircuitCooldown         = 2 * time.Minute
	openAIModelLatencyCircuitHighCooldown     = 5 * time.Minute
	openAIModelLatencyCircuitProbeTimeout     = 2 * time.Minute
	openAIModelLatencyCircuitMaxEntries       = 4096
)

type openAIAccountModelLatencyCircuitEntry struct {
	failureCount  int
	windowStart   time.Time
	blockUntil    time.Time
	probeInFlight bool
	probeUntil    time.Time
	cooldown      time.Duration
	lastTouched   time.Time
}

type openAIAccountModelLatencyCircuitDecision struct {
	FailureCount int
	BlockUntil   time.Time
	Cooldown     time.Duration
}

type openAIAccountModelLatencyCircuit struct {
	mu         sync.Mutex
	entries    map[openAIAccountModelKey]openAIAccountModelLatencyCircuitEntry
	maxEntries int
}

func newOpenAIAccountModelLatencyCircuit(maxEntries int) *openAIAccountModelLatencyCircuit {
	if maxEntries <= 0 {
		maxEntries = openAIModelLatencyCircuitMaxEntries
	}
	return &openAIAccountModelLatencyCircuit{
		entries:    make(map[openAIAccountModelKey]openAIAccountModelLatencyCircuitEntry),
		maxEntries: maxEntries,
	}
}

func (s *openAIAccountModelLatencyCircuit) recordTimeout(accountID int64, model, reasoningEffort string, now time.Time) openAIAccountModelLatencyCircuitDecision {
	key, ok := openAIAccountModelTransientKey(accountID, model)
	if s == nil || !ok {
		return openAIAccountModelLatencyCircuitDecision{}
	}
	if now.IsZero() {
		now = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = make(map[openAIAccountModelKey]openAIAccountModelLatencyCircuitEntry)
	}
	entry, exists := s.entries[key]
	if !exists {
		s.evictOldestLocked()
	}
	if !exists || entry.windowStart.IsZero() || now.Before(entry.windowStart) || now.Sub(entry.windowStart) > openAIModelLatencyCircuitWindow {
		entry.failureCount = 0
		entry.windowStart = now
	}
	entry.failureCount++
	entry.lastTouched = now
	entry.probeInFlight = false
	entry.probeUntil = time.Time{}

	cooldown := openAIModelLatencyCircuitCooldown
	switch strings.ToLower(strings.TrimSpace(reasoningEffort)) {
	case "high", "xhigh", "max":
		cooldown = openAIModelLatencyCircuitHighCooldown
	}
	entry.cooldown = cooldown
	if entry.failureCount >= openAIModelLatencyCircuitFailureThreshold {
		entry.blockUntil = now.Add(cooldown)
	}
	s.entries[key] = entry
	return openAIAccountModelLatencyCircuitDecision{
		FailureCount: entry.failureCount,
		BlockUntil:   entry.blockUntil,
		Cooldown:     cooldown,
	}
}

func (s *openAIAccountModelLatencyCircuit) isBlocked(accountID int64, model string, now time.Time) bool {
	key, ok := openAIAccountModelTransientKey(accountID, model)
	if s == nil || !ok {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.entries[key]
	if !exists {
		return false
	}
	entry.lastTouched = now
	if entry.probeInFlight && !entry.probeUntil.IsZero() && !now.Before(entry.probeUntil) {
		entry.probeInFlight = false
		entry.probeUntil = time.Time{}
	}
	s.entries[key] = entry
	return now.Before(entry.blockUntil) || entry.probeInFlight
}

func (s *openAIAccountModelLatencyCircuit) tryClaimProbe(accountID int64, model string, now time.Time) bool {
	key, ok := openAIAccountModelTransientKey(accountID, model)
	if s == nil || !ok {
		return true
	}
	if now.IsZero() {
		now = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.entries[key]
	if !exists || entry.blockUntil.IsZero() {
		return true
	}
	if now.Before(entry.blockUntil) {
		return false
	}
	if entry.probeInFlight && now.Before(entry.probeUntil) {
		return false
	}
	entry.probeInFlight = true
	entry.probeUntil = now.Add(openAIModelLatencyCircuitProbeTimeout)
	entry.lastTouched = now
	s.entries[key] = entry
	return true
}

func (s *openAIAccountModelLatencyCircuit) recordResult(accountID int64, model string, success bool, now time.Time) {
	key, ok := openAIAccountModelTransientKey(accountID, model)
	if s == nil || !ok {
		return
	}
	if now.IsZero() {
		now = time.Now()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.entries[key]
	if !exists {
		return
	}
	if success {
		delete(s.entries, key)
		return
	}
	if entry.probeInFlight {
		cooldown := entry.cooldown
		if cooldown <= 0 {
			cooldown = openAIModelLatencyCircuitCooldown
		}
		entry.probeInFlight = false
		entry.probeUntil = time.Time{}
		entry.blockUntil = now.Add(cooldown)
		entry.windowStart = now
		entry.failureCount = openAIModelLatencyCircuitFailureThreshold
		entry.lastTouched = now
		s.entries[key] = entry
	}
}

func (s *openAIAccountModelLatencyCircuit) evictOldestLocked() {
	if len(s.entries) < s.maxEntries {
		return
	}
	var oldestKey openAIAccountModelKey
	var oldestTime time.Time
	found := false
	for key, entry := range s.entries {
		if !found || entry.lastTouched.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.lastTouched
			found = true
		}
	}
	if found {
		delete(s.entries, oldestKey)
	}
}
