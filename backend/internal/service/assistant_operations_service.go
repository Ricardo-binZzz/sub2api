package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type AssistantOperationsService struct {
	repo      OperationRepository
	encryptor SecretEncryptor
	assistant *AssistantService
	cfg       *config.Config
	http      *http.Client
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mu        sync.Mutex
	now       func() time.Time
}

type OperationSummary struct {
	Policy         *OperationPolicy `json:"policy"`
	Connections    int              `json:"connections"`
	PendingTasks   int64            `json:"pending_tasks"`
	SucceededToday int              `json:"succeeded_today"`
}

func NewAssistantOperationsService(repo OperationRepository, encryptor SecretEncryptor, assistant *AssistantService, cfg *config.Config) *AssistantOperationsService {
	return &AssistantOperationsService{
		repo: repo, encryptor: encryptor, assistant: assistant, cfg: cfg,
		http: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return errors.New("external platform redirect rejected")
			},
		},
		now: time.Now,
	}
}
func ProvideAssistantOperationsService(repo OperationRepository, encryptor SecretEncryptor, assistant *AssistantService, cfg *config.Config) *AssistantOperationsService {
	s := NewAssistantOperationsService(repo, encryptor, assistant, cfg)
	s.Start()
	return s
}
func (s *AssistantOperationsService) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.wg.Add(1)
	go s.loop(ctx)
}
func (s *AssistantOperationsService) Stop() {
	s.mu.Lock()
	cancel := s.cancel
	s.cancel = nil
	s.mu.Unlock()
	if cancel != nil {
		cancel()
		s.wg.Wait()
	}
}
func (s *AssistantOperationsService) loop(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.tick(ctx)
		}
	}
}
func (s *AssistantOperationsService) tick(ctx context.Context) {
	policy, err := s.repo.GetPolicy(ctx)
	if err != nil || !policy.Enabled {
		return
	}
	if policy.AutonomousEnabled {
		_ = s.PlanNow(ctx)
	}
	ids, err := s.repo.ListDueTaskIDs(ctx, 10)
	if err == nil {
		for _, id := range ids {
			_ = s.RunTask(ctx, id)
		}
	}
	s.pollTelegram(ctx)
}

func (s *AssistantOperationsService) Summary(ctx context.Context) (*OperationSummary, error) {
	p, e := s.repo.GetPolicy(ctx)
	if e != nil {
		return nil, e
	}
	c, e := s.repo.ListConnections(ctx)
	if e != nil {
		return nil, e
	}
	pending, e := s.repo.CountPendingTasks(ctx)
	if e != nil {
		return nil, e
	}
	today := s.now().UTC().Truncate(24 * time.Hour)
	n, e := s.repo.CountSucceededSince(ctx, today)
	if e != nil {
		return nil, e
	}
	return &OperationSummary{Policy: p, Connections: len(c), PendingTasks: pending, SucceededToday: n}, nil
}
func (s *AssistantOperationsService) ListConnections(ctx context.Context) ([]*OperationConnection, error) {
	rows, e := s.repo.ListConnections(ctx)
	if e != nil {
		return nil, e
	}
	out := make([]*OperationConnection, 0, len(rows))
	for _, r := range rows {
		out = append(out, &r.OperationConnection)
	}
	return out, nil
}
func (s *AssistantOperationsService) SaveConnection(ctx context.Context, id int64, in OperationConnectionInput) (*OperationConnection, error) {
	in.Platform = strings.ToLower(strings.TrimSpace(in.Platform))
	in.Name = strings.TrimSpace(in.Name)
	if !validPlatform(in.Platform) || in.Name == "" {
		return nil, ErrOperationInvalid
	}
	configMap := in.Config
	if id > 0 {
		old, e := s.repo.GetConnection(ctx, id)
		if e != nil {
			return nil, e
		}
		if len(configMap) == 0 {
			if old.Platform != in.Platform {
				return nil, ErrOperationInvalid
			}
			old.Name = in.Name
			old.Enabled = in.Enabled
			saved, e := s.repo.SaveConnection(ctx, old)
			if e != nil {
				return nil, e
			}
			return &saved.OperationConnection, nil
		}
	}
	if e := validateConnectionConfig(in.Platform, configMap); e != nil {
		return nil, e
	}
	raw, e := json.Marshal(configMap)
	if e != nil {
		return nil, e
	}
	encrypted, e := s.encryptor.Encrypt(string(raw))
	if e != nil {
		return nil, e
	}
	saved, e := s.repo.SaveConnection(ctx, &OperationConnectionRecord{OperationConnection: OperationConnection{ID: id, Platform: in.Platform, Name: in.Name, Enabled: in.Enabled}, EncryptedConfig: encrypted})
	if e != nil {
		return nil, e
	}
	return &saved.OperationConnection, nil
}
func (s *AssistantOperationsService) DeleteConnection(ctx context.Context, id int64) error {
	return s.repo.DeleteConnection(ctx, id)
}
func (s *AssistantOperationsService) GetPolicy(ctx context.Context) (*OperationPolicy, error) {
	return s.repo.GetPolicy(ctx)
}
func (s *AssistantOperationsService) UpdatePolicy(ctx context.Context, p OperationPolicy) (*OperationPolicy, error) {
	if p.PlanningIntervalMinutes < 30 || p.PlanningIntervalMinutes > 10080 || p.MinPublishIntervalMinutes < 15 || p.MinPublishIntervalMinutes > 10080 || p.MaxDailyActions < 0 || p.MaxDailyActions > 50 || p.QuietHoursStart < 0 || p.QuietHoursStart > 23 || p.QuietHoursEnd < 0 || p.QuietHoursEnd > 23 {
		return nil, ErrOperationInvalid
	}
	return s.repo.UpdatePolicy(ctx, p)
}
func (s *AssistantOperationsService) ListTasks(ctx context.Context, page, size int) ([]*OperationTask, int64, error) {
	return s.repo.ListTasks(ctx, page, size)
}
func (s *AssistantOperationsService) ListRuns(ctx context.Context, page, size int) ([]*OperationRun, int64, error) {
	return s.repo.ListRuns(ctx, page, size)
}
func (s *AssistantOperationsService) CreateTask(ctx context.Context, connectionID int64, kind, content string, scheduled time.Time) (*OperationTask, error) {
	conn, e := s.repo.GetConnection(ctx, connectionID)
	if e != nil {
		return nil, e
	}
	kind = strings.TrimSpace(kind)
	content = strings.TrimSpace(content)
	if kind != "promotion" && kind != "notification" || content == "" || len([]rune(content)) > 3000 {
		return nil, ErrOperationInvalid
	}
	if scheduled.IsZero() {
		scheduled = s.now().UTC()
	}
	p, e := s.repo.GetPolicy(ctx)
	if e != nil {
		return nil, e
	}
	status := "pending_approval"
	if !p.RequireApproval {
		status = "approved"
	}
	key := operationKey(conn.Platform, content, scheduled)
	return s.repo.CreateTask(ctx, OperationTaskInput{ConnectionID: connectionID, Kind: kind, Platform: conn.Platform, Status: status, Content: content, IdempotencyKey: key, ScheduledAt: scheduled})
}
func (s *AssistantOperationsService) ApproveTask(ctx context.Context, id, userID int64) (*OperationTask, error) {
	return s.repo.ApproveTask(ctx, id, userID)
}
func (s *AssistantOperationsService) CancelTask(ctx context.Context, id int64) error {
	return s.repo.CancelTask(ctx, id)
}
func (s *AssistantOperationsService) TestConnection(ctx context.Context, id int64) error {
	conn, e := s.repo.GetConnection(ctx, id)
	if e != nil {
		return e
	}
	cfg, e := s.connectionConfig(conn)
	if e == nil {
		e = s.testPlatformConnection(ctx, conn, cfg)
	}
	msg := externalErrorMessage(e)
	_ = s.repo.UpdateConnectionHealth(ctx, id, msg, e == nil)
	return e
}

func (s *AssistantOperationsService) PlanNow(ctx context.Context) error {
	p, e := s.repo.GetPolicy(ctx)
	if e != nil {
		return e
	}
	if !p.Enabled || !p.AutonomousEnabled {
		return ErrOperationPolicyBlocked
	}
	last, e := s.repo.LastTaskCreatedAt(ctx, "promotion")
	if e != nil {
		return e
	}
	if last != nil && s.now().Sub(*last) < time.Duration(p.PlanningIntervalMinutes)*time.Minute {
		return nil
	}
	conns, e := s.repo.ListConnections(ctx)
	if e != nil {
		return e
	}
	var target *OperationConnectionRecord
	for _, c := range conns {
		if c.Enabled && c.Platform == "bluesky" {
			target = c
			break
		}
	}
	if target == nil {
		return nil
	}
	reply, e := s.assistant.ChatAdmin(ctx, "根据当前中转站最近24小时运营数据，写一条自然、克制、真实的中文宣传短帖。只输出可直接发布的正文，不要标题、引号、解释、标签说明或AI口吻，最多260字。", nil)
	if e != nil {
		return e
	}
	status := "pending_approval"
	if p.AutoPublish && !p.RequireApproval {
		status = "approved"
	}
	now := s.now().UTC()
	_, e = s.repo.CreateTask(ctx, OperationTaskInput{ConnectionID: target.ID, Kind: "promotion", Platform: target.Platform, Status: status, Content: strings.TrimSpace(reply.Answer), IdempotencyKey: operationKey(target.Platform, reply.Answer, now), ScheduledAt: now})
	return e
}

func (s *AssistantOperationsService) RunTask(ctx context.Context, id int64) error {
	p, e := s.repo.GetPolicy(ctx)
	if e != nil {
		return e
	}
	task, e := s.repo.GetTask(ctx, id)
	if e != nil {
		return e
	}
	deferUntil, e := s.executionPolicyDeferral(ctx, p, task)
	if e != nil {
		if deferUntil != nil {
			message := externalErrorMessage(e)
			if deferErr := s.repo.DeferTask(ctx, id, *deferUntil, message); deferErr != nil {
				return deferErr
			}
		}
		return e
	}
	task, runID, e := s.repo.BeginTaskRun(ctx, id)
	if e != nil {
		return e
	}
	var external string
	conn, ce := s.repo.GetConnection(ctx, derefID(task.ConnectionID))
	if ce != nil {
		e = ce
	} else if !conn.Enabled {
		e = ErrOperationPolicyBlocked
	} else {
		cfg, ce := s.connectionConfig(conn)
		if ce != nil {
			e = ce
		} else {
			external, e = s.deliver(ctx, conn, cfg, task.Content)
		}
		msg := externalErrorMessage(e)
		_ = s.repo.UpdateConnectionHealth(ctx, conn.ID, msg, e == nil)
	}
	retry := s.now().UTC().Add(time.Duration(1<<min(task.Attempts, 6)) * time.Minute)
	message := "delivered"
	if e != nil {
		message = externalErrorMessage(e)
	}
	completeErr := s.repo.CompleteTask(ctx, id, runID, external, message, e == nil, retry)
	if completeErr != nil {
		return completeErr
	}
	return e
}
func (s *AssistantOperationsService) checkExecutionPolicy(ctx context.Context, p *OperationPolicy, t *OperationTask) error {
	_, err := s.executionPolicyDeferral(ctx, p, t)
	return err
}

func (s *AssistantOperationsService) executionPolicyDeferral(ctx context.Context, p *OperationPolicy, t *OperationTask) (*time.Time, error) {
	if !p.Enabled {
		return nil, ErrOperationPolicyBlocked
	}
	now := s.now()
	hour := now.Hour()
	if inQuietHours(hour, p.QuietHoursStart, p.QuietHoursEnd) {
		resume := quietHoursEnd(now, p.QuietHoursStart, p.QuietHoursEnd)
		return &resume, fmt.Errorf("%w: quiet hours", ErrOperationPolicyBlocked)
	}
	n, e := s.repo.CountSucceededSince(ctx, now.UTC().Truncate(24*time.Hour))
	if e != nil {
		return nil, e
	}
	if n >= p.MaxDailyActions {
		resume := now.UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		return &resume, fmt.Errorf("%w: daily budget exhausted", ErrOperationPolicyBlocked)
	}
	last, e := s.repo.LastSucceededAt(ctx, t.Kind)
	if e != nil {
		return nil, e
	}
	if last != nil && t.Kind == "promotion" && now.Sub(*last) < time.Duration(p.MinPublishIntervalMinutes)*time.Minute {
		resume := last.Add(time.Duration(p.MinPublishIntervalMinutes) * time.Minute)
		return &resume, fmt.Errorf("%w: publish interval", ErrOperationPolicyBlocked)
	}
	return nil, nil
}

func quietHoursEnd(now time.Time, start, end int) time.Time {
	resume := time.Date(now.Year(), now.Month(), now.Day(), end, 0, 0, 0, now.Location())
	if start > end && now.Hour() >= start || start == end || !resume.After(now) {
		resume = resume.Add(24 * time.Hour)
	}
	return resume
}

func (s *AssistantOperationsService) connectionConfig(c *OperationConnectionRecord) (map[string]string, error) {
	plain, e := s.encryptor.Decrypt(c.EncryptedConfig)
	if e != nil {
		return nil, e
	}
	var m map[string]string
	if e = json.Unmarshal([]byte(plain), &m); e != nil {
		return nil, e
	}
	return m, nil
}
func (s *AssistantOperationsService) deliver(ctx context.Context, c *OperationConnectionRecord, cfg map[string]string, content string) (string, error) {
	switch c.Platform {
	case "bluesky":
		return s.postBluesky(ctx, cfg, content)
	case "telegram":
		return s.sendTelegram(ctx, cfg, content)
	case "discord":
		return s.sendDiscord(ctx, cfg, content)
	default:
		return "", ErrOperationInvalid
	}
}
func (s *AssistantOperationsService) testPlatformConnection(ctx context.Context, c *OperationConnectionRecord, cfg map[string]string) error {
	switch c.Platform {
	case "bluesky":
		var out struct {
			DID string `json:"did"`
		}
		return s.jsonRequest(ctx, http.MethodPost, "https://bsky.social/xrpc/com.atproto.server.createSession", map[string]string{
			"identifier": cfg["identifier"], "password": cfg["app_password"],
		}, "", &out)
	case "telegram":
		var out map[string]any
		return s.jsonRequest(ctx, http.MethodGet, telegramEndpoint(cfg["bot_token"], "getMe"), nil, "", &out)
	case "discord":
		raw, err := validatePlatformURL(cfg["webhook_url"], s.cfg, "discord.com", "discordapp.com")
		if err != nil {
			return err
		}
		return s.jsonRequest(ctx, http.MethodGet, raw, nil, "", nil)
	default:
		return ErrOperationInvalid
	}
}
func (s *AssistantOperationsService) postBluesky(ctx context.Context, cfg map[string]string, text string) (string, error) {
	type session struct {
		DID    string `json:"did"`
		Access string `json:"accessJwt"`
	}
	var ses session
	e := s.jsonRequest(ctx, http.MethodPost, "https://bsky.social/xrpc/com.atproto.server.createSession", map[string]string{"identifier": cfg["identifier"], "password": cfg["app_password"]}, "", &ses)
	if e != nil {
		return "", e
	}
	payload := map[string]any{"repo": ses.DID, "collection": "app.bsky.feed.post", "record": map[string]any{"$type": "app.bsky.feed.post", "text": text, "createdAt": s.now().UTC().Format(time.RFC3339)}}
	var out struct {
		URI string `json:"uri"`
	}
	e = s.jsonRequest(ctx, http.MethodPost, "https://bsky.social/xrpc/com.atproto.repo.createRecord", payload, ses.Access, &out)
	return out.URI, e
}
func (s *AssistantOperationsService) sendTelegram(ctx context.Context, cfg map[string]string, text string) (string, error) {
	return s.sendTelegramTo(ctx, cfg["bot_token"], cfg["chat_id"], text)
}
func (s *AssistantOperationsService) sendTelegramTo(ctx context.Context, token, chatID, text string) (string, error) {
	endpoint := telegramEndpoint(token, "sendMessage")
	var out struct {
		OK     bool `json:"ok"`
		Result struct {
			MessageID int `json:"message_id"`
		} `json:"result"`
	}
	e := s.jsonRequest(ctx, http.MethodPost, endpoint, map[string]string{"chat_id": chatID, "text": text}, "", &out)
	if e == nil && !out.OK {
		e = errors.New("telegram rejected message")
	}
	return strconv.Itoa(out.Result.MessageID), e
}
func (s *AssistantOperationsService) sendDiscord(ctx context.Context, cfg map[string]string, text string) (string, error) {
	raw, e := validatePlatformURL(cfg["webhook_url"], s.cfg, "discord.com", "discordapp.com")
	if e != nil {
		return "", e
	}
	e = s.jsonRequest(ctx, http.MethodPost, raw, map[string]string{"content": text}, "", nil)
	return "discord-webhook", e
}
func (s *AssistantOperationsService) jsonRequest(ctx context.Context, method, raw string, payload any, bearer string, out any) error {
	var body io.Reader
	if payload != nil {
		encoded, e := json.Marshal(payload)
		if e != nil {
			return errors.New("external platform request encoding failed")
		}
		body = bytes.NewReader(encoded)
	}
	req, e := http.NewRequestWithContext(ctx, method, raw, body)
	if e != nil {
		return errors.New("external platform request is invalid")
	}
	req.Header.Set("Content-Type", "application/json")
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, e := s.http.Do(req)
	if e != nil {
		return errors.New("external platform request failed")
	}
	defer func() { _ = resp.Body.Close() }()
	data, e := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if e != nil {
		return errors.New("external platform response could not be read")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("external platform returned HTTP %d", resp.StatusCode)
	}
	if out != nil && len(data) > 0 {
		if e := json.Unmarshal(data, out); e != nil {
			return errors.New("external platform returned an invalid response")
		}
	}
	return nil
}

type telegramResponse struct {
	OK     bool             `json:"ok"`
	Result []telegramUpdate `json:"result"`
}
type telegramUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Text string `json:"text"`
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
	} `json:"message"`
}

func (s *AssistantOperationsService) pollTelegram(ctx context.Context) {
	connections, err := s.repo.ListConnections(ctx)
	if err != nil {
		return
	}
	for _, conn := range connections {
		if !conn.Enabled || conn.Platform != "telegram" {
			continue
		}
		cfg, err := s.connectionConfig(conn)
		if err != nil {
			continue
		}
		allowed := parseChatIDs(cfg["admin_chat_ids"])
		if len(allowed) == 0 {
			continue
		}
		offset, _ := strconv.ParseInt(conn.LastCursor, 10, 64)
		endpoint := telegramEndpoint(cfg["bot_token"], "getUpdates") + "?timeout=0&limit=25&offset=" + strconv.FormatInt(offset+1, 10)
		var response telegramResponse
		if err := s.jsonRequest(ctx, http.MethodGet, endpoint, nil, "", &response); err != nil || !response.OK {
			_ = s.repo.UpdateConnectionHealth(ctx, conn.ID, "messaging poll failed", false)
			continue
		}
		var maxUpdateID int64
		for _, update := range response.Result {
			if update.UpdateID > maxUpdateID {
				maxUpdateID = update.UpdateID
			}
			if update.Message != nil {
				chatID := strconv.FormatInt(update.Message.Chat.ID, 10)
				if allowed[chatID] {
					s.handleTelegramCommand(ctx, cfg["bot_token"], chatID, update.Message.Text)
				}
			}
		}
		if maxUpdateID > 0 {
			_ = s.repo.UpdateConnectionCursor(ctx, conn.ID, strconv.FormatInt(maxUpdateID, 10))
		}
		_ = s.repo.UpdateConnectionHealth(ctx, conn.ID, "", true)
	}
}

func (s *AssistantOperationsService) handleTelegramCommand(ctx context.Context, token, chatID, input string) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) == 0 {
		return
	}
	command := strings.ToLower(strings.SplitN(fields[0], "@", 2)[0])
	message := "Unknown command. Use /status, /pause, /resume, or /approve ID."
	switch command {
	case "/status":
		summary, err := s.Summary(ctx)
		if err == nil {
			message = fmt.Sprintf("AI operations: enabled=%t autonomous=%t pending=%d succeeded_today=%d", summary.Policy.Enabled, summary.Policy.AutonomousEnabled, summary.PendingTasks, summary.SucceededToday)
		} else {
			message = "Status is temporarily unavailable."
		}
	case "/pause":
		if err := s.repo.SetAutonomousEnabled(ctx, false); err == nil {
			message = "Autonomous operations paused."
		} else {
			message = "Pause failed."
		}
	case "/resume":
		policy, err := s.repo.GetPolicy(ctx)
		if err == nil && policy.Enabled {
			err = s.repo.SetAutonomousEnabled(ctx, true)
		}
		if err == nil && policy != nil && policy.Enabled {
			message = "Autonomous operations resumed."
		} else {
			message = "Enable the operations runtime in the admin console first."
		}
	case "/approve":
		if len(fields) == 2 {
			id, err := strconv.ParseInt(fields[1], 10, 64)
			if err == nil && id > 0 {
				_, err = s.repo.ApproveTask(ctx, id, 0)
			}
			if err == nil && id > 0 {
				message = fmt.Sprintf("Task %d approved.", id)
			} else {
				message = "Task approval failed."
			}
		} else {
			message = "Usage: /approve ID"
		}
	}
	_, _ = s.sendTelegramTo(ctx, token, chatID, message)
}

var telegramTokenPattern = regexp.MustCompile(`^[0-9]{6,16}:[A-Za-z0-9_-]{20,80}$`)

func telegramEndpoint(token, method string) string {
	return "https://api.telegram.org/bot" + token + "/" + method
}

func parseChatIDs(value string) map[string]bool {
	result := make(map[string]bool)
	for _, item := range strings.Split(value, ",") {
		item = strings.TrimSpace(item)
		if _, err := strconv.ParseInt(item, 10, 64); err == nil && item != "0" {
			result[item] = true
		}
	}
	return result
}

func validateConnectionConfig(platform string, m map[string]string) error {
	required := map[string][]string{"bluesky": {"identifier", "app_password"}, "telegram": {"bot_token", "chat_id", "admin_chat_ids"}, "discord": {"webhook_url"}}[platform]
	for _, k := range required {
		if strings.TrimSpace(m[k]) == "" {
			return fmt.Errorf("%w: missing %s", ErrOperationInvalid, k)
		}
	}
	if platform == "telegram" && (!telegramTokenPattern.MatchString(strings.TrimSpace(m["bot_token"])) || len(parseChatIDs(m["admin_chat_ids"])) == 0) {
		return ErrOperationInvalid
	}
	if platform == "discord" {
		u, err := url.Parse(strings.TrimSpace(m["webhook_url"]))
		if err != nil || u.Scheme != "https" || (u.Hostname() != "discord.com" && u.Hostname() != "discordapp.com") || !strings.HasPrefix(u.EscapedPath(), "/api/webhooks/") {
			return ErrOperationInvalid
		}
	}
	return nil
}
func validatePlatformURL(raw string, cfg *config.Config, hosts ...string) (string, error) {
	v, e := validateOutboundURL(raw, cfg, hosts)
	if e != nil {
		return "", e
	}
	u, e := url.Parse(v)
	if e != nil {
		return "", ErrOperationInvalid
	}
	for _, host := range hosts {
		if strings.EqualFold(u.Hostname(), host) {
			return v, nil
		}
	}
	return "", ErrOperationInvalid
}
func externalErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, ErrOperationPolicyBlocked) || errors.Is(err, ErrOperationInvalid) || errors.Is(err, ErrOperationNotFound) {
		return err.Error()
	}
	return "external platform operation failed"
}
func validPlatform(v string) bool { return v == "bluesky" || v == "telegram" || v == "discord" }
func operationKey(platform, content string, t time.Time) string {
	sum := sha256.Sum256([]byte(platform + "\x00" + content + "\x00" + t.UTC().Format("2006-01-02T15:04")))
	return platform + ":" + hex.EncodeToString(sum[:])
}
func inQuietHours(hour, start, end int) bool {
	if start == end {
		return false
	}
	if start < end {
		return hour >= start && hour < end
	}
	return hour >= start || hour < end
}
func derefID(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}
