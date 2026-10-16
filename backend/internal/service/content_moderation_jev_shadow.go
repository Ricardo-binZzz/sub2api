package service

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"log/slog"
	"time"
)

type jevShadowTask struct {
	RequestID       string
	Text            string
	CurrentFlagged  bool
	CurrentCategory string
	CurrentScore    float64
}

// Best-effort legacy/new comparison at the same existing JEV provider. No new
// data destination, no stored prompts, no sanctions or blocklist side effects.
func (s *ContentModerationService) enqueueJevShadow(input ContentModerationCheckInput, cfg *ContentModerationConfig, current *moderationAPIResult, inputHash string) {
	if s.asyncQueue == nil || len(s.asyncQueue) >= 16 || cfg.Engine != ContentModerationEngineTypeSafe || cfg.Mode != ContentModerationModePreBlock {
		return
	}
	sum := sha256.Sum256([]byte(input.RequestID + "\x00" + inputHash))
	if binary.BigEndian.Uint64(sum[:8])%100 != 0 {
		return
	}
	now := time.Now().UnixNano()
	old := s.jevShadowLast.Load()
	if now-old < int64(10*time.Second) || !s.jevShadowLast.CompareAndSwap(old, now) {
		return
	}
	baseline := ExtractContentModerationInput(input.Protocol, input.Body)
	if baseline.Text == "" {
		return
	}
	flagged, category, score := evaluateModerationScores(current.CategoryScores, cfg.Thresholds)
	task := contentModerationTask{config: cloneContentModerationConfig(cfg), jevShadow: &jevShadowTask{input.RequestID, baseline.Text, flagged, category, score}, enqueuedAt: time.Now()}
	select {
	case s.asyncQueue <- task:
	default:
	}
}

func (s *ContentModerationService) runJevShadow(ctx context.Context, task contentModerationTask) {
	if task.jevShadow == nil || task.config == nil || s.preBlockActive.Load() > 0 || time.Since(task.enqueuedAt) > 5*time.Second || !s.jevShadowActive.CompareAndSwap(false, true) {
		return
	}
	defer s.jevShadowActive.Store(false)
	cfg := cloneContentModerationConfig(task.config)
	cfg.typesafeLegacyRules = true
	cfg.RetryCount = 0
	if cfg.TimeoutMS > 1000 {
		cfg.TimeoutMS = 1000
	}
	started := time.Now()
	key, available := s.nextUsableAPIKey(cfg)
	if !available {
		return
	}
	// Shadow errors must not freeze/penalize a key used by foreground admission.
	result, err := s.callTypeSafeModeration(ctx, cfg, key, task.jevShadow.Text, nil)
	if err != nil {
		slog.Info("content_moderation.shadow_comparison", "request_id", task.jevShadow.RequestID, "status", "unavailable", "latency_ms", time.Since(started).Milliseconds())
		return
	}
	flagged, category, score := evaluateModerationScores(result.CategoryScores, cfg.Thresholds)
	slog.Info("content_moderation.shadow_comparison", "request_id", task.jevShadow.RequestID, "status", "compared", "current_rules", TypeSafeModerationRulesVersion, "baseline_rules", "content-audit-13-zh-v1", "disagrees", flagged != task.jevShadow.CurrentFlagged, "current_flagged", task.jevShadow.CurrentFlagged, "baseline_flagged", flagged, "current_category", task.jevShadow.CurrentCategory, "baseline_category", category, "current_score", task.jevShadow.CurrentScore, "baseline_score", score, "latency_ms", time.Since(started).Milliseconds())
}
