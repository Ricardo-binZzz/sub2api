package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

// GetUpstreamPerformance returns read-only latency aggregates for groups and accounts.
func (r *usageLogRepository) GetUpstreamPerformance(ctx context.Context, startTime, endTime time.Time) (*service.UpstreamPerformanceReport, error) {
	metricSelect := fmt.Sprintf(`
COUNT(*) AS requests,
COUNT(*) FILTER (WHERE %s) AS successful,
percentile_cont(0.50) WITHIN GROUP (ORDER BY ul.first_token_ms) FILTER (WHERE ul.first_token_ms IS NOT NULL AND %s),
percentile_cont(0.95) WITHIN GROUP (ORDER BY ul.first_token_ms) FILTER (WHERE ul.first_token_ms IS NOT NULL AND %s),
percentile_cont(0.50) WITHIN GROUP (ORDER BY ul.duration_ms) FILTER (WHERE ul.duration_ms IS NOT NULL AND %s),
percentile_cont(0.95) WITHIN GROUP (ORDER BY ul.duration_ms) FILTER (WHERE ul.duration_ms IS NOT NULL AND %s)`,
		usageLogSuccessFilterUL,
		usageLogSuccessFilterUL,
		usageLogSuccessFilterUL,
		usageLogSuccessFilterUL,
		usageLogSuccessFilterUL,
	)

	query := fmt.Sprintf(`
SELECT 'group' AS entity_type, ul.group_id AS entity_id,
       COALESCE(NULLIF(g.name, ''), 'Unassigned') AS name,
       COALESCE(NULLIF(g.platform, ''), '') AS platform,
       %s
FROM usage_logs ul
LEFT JOIN groups g ON g.id = ul.group_id
WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.group_id IS NOT NULL
GROUP BY ul.group_id, g.name, g.platform
UNION ALL
SELECT 'account' AS entity_type, ul.account_id AS entity_id,
       COALESCE(NULLIF(a.name, ''), 'Unknown account') AS name,
       COALESCE(NULLIF(a.platform, ''), '') AS platform,
       %s
FROM usage_logs ul
LEFT JOIN accounts a ON a.id = ul.account_id
WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.account_id IS NOT NULL
GROUP BY ul.account_id, a.name, a.platform
ORDER BY entity_type, requests DESC, entity_id`, metricSelect, metricSelect)

	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = rows.Close()
	}()

	report := &service.UpstreamPerformanceReport{
		StartTime:   startTime,
		EndTime:     endTime,
		GeneratedAt: time.Now().UTC(),
		Groups:      make([]service.UpstreamPerformanceMetric, 0),
		Accounts:    make([]service.UpstreamPerformanceMetric, 0),
		Thresholds: service.UpstreamPerformanceThresholds{
			FirstTokenP50WarningMs: 3000,
			FirstTokenP95WarningMs: 8000,
			SuccessRateWarning:     98,
		},
	}
	for rows.Next() {
		var entityType string
		var metric service.UpstreamPerformanceMetric
		var firstP50, firstP95, durationP50, durationP95 *float64
		if err := rows.Scan(
			&entityType,
			&metric.ID,
			&metric.Name,
			&metric.Platform,
			&metric.Requests,
			&metric.Successful,
			&firstP50,
			&firstP95,
			&durationP50,
			&durationP95,
		); err != nil {
			return nil, err
		}
		if metric.Requests > 0 {
			metric.SuccessRate = float64(metric.Successful) * 100 / float64(metric.Requests)
		}
		metric.FirstTokenP50Ms = firstP50
		metric.FirstTokenP95Ms = firstP95
		metric.DurationP50Ms = durationP50
		metric.DurationP95Ms = durationP95
		if entityType == "group" {
			report.Groups = append(report.Groups, metric)
		} else {
			report.Accounts = append(report.Accounts, metric)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return report, nil
}
