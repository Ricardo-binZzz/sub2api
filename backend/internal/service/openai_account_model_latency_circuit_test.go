package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenAIAccountModelLatencyCircuit_TripsPerModelAndAllowsSingleProbe(t *testing.T) {
	circuit := newOpenAIAccountModelLatencyCircuit(16)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)

	first := circuit.recordTimeout(1001, "gpt-slow", "medium", now)
	require.Equal(t, 1, first.FailureCount)
	require.True(t, first.BlockUntil.IsZero())
	require.False(t, circuit.isBlocked(1001, "gpt-slow", now))

	second := circuit.recordTimeout(1001, "gpt-slow", "medium", now.Add(10*time.Second))
	require.Equal(t, openAIModelLatencyCircuitCooldown, second.Cooldown)
	require.Equal(t, now.Add(10*time.Second).Add(openAIModelLatencyCircuitCooldown), second.BlockUntil)
	require.True(t, circuit.isBlocked(1001, "gpt-slow", now.Add(time.Minute)))
	require.False(t, circuit.isBlocked(1001, "gpt-fast", now.Add(time.Minute)))

	probeAt := second.BlockUntil.Add(time.Second)
	require.True(t, circuit.tryClaimProbe(1001, "gpt-slow", probeAt))
	require.True(t, circuit.isBlocked(1001, "gpt-slow", probeAt))
	require.False(t, circuit.tryClaimProbe(1001, "gpt-slow", probeAt))

	circuit.recordResult(1001, "gpt-slow", true, probeAt.Add(time.Second))
	require.False(t, circuit.isBlocked(1001, "gpt-slow", probeAt.Add(2*time.Second)))
}

func TestOpenAIAccountModelLatencyCircuit_HighEffortUsesLongCooldown(t *testing.T) {
	circuit := newOpenAIAccountModelLatencyCircuit(16)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	circuit.recordTimeout(1001, "gpt-high", "xhigh", now)
	decision := circuit.recordTimeout(1001, "gpt-high", "xhigh", now.Add(10*time.Second))

	require.Equal(t, openAIModelLatencyCircuitHighCooldown, decision.Cooldown)
	require.Equal(t, now.Add(10*time.Second).Add(openAIModelLatencyCircuitHighCooldown), decision.BlockUntil)
}

func TestOpenAIAccountModelLatencyCircuit_FailedProbeReopens(t *testing.T) {
	circuit := newOpenAIAccountModelLatencyCircuit(16)
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	circuit.recordTimeout(1001, "gpt-slow", "medium", now)
	decision := circuit.recordTimeout(1001, "gpt-slow", "medium", now.Add(time.Second))
	probeAt := decision.BlockUntil.Add(time.Second)
	require.True(t, circuit.tryClaimProbe(1001, "gpt-slow", probeAt))

	circuit.recordResult(1001, "gpt-slow", false, probeAt.Add(time.Second))
	require.True(t, circuit.isBlocked(1001, "gpt-slow", probeAt.Add(2*time.Second)))
}
