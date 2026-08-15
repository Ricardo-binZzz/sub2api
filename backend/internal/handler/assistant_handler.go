package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AssistantHandler struct {
	assistant  *service.AssistantService
	operations *service.AssistantOperationsService
	audit      *service.AuditLogService
}

func NewAssistantHandler(assistant *service.AssistantService, operations *service.AssistantOperationsService, audit *service.AuditLogService) *AssistantHandler {
	return &AssistantHandler{assistant: assistant, operations: operations, audit: audit}
}

type assistantChatRequest struct {
	Question string                     `json:"question" binding:"required"`
	History  []service.AssistantMessage `json:"history"`
}

type assistantRewardApplyResponse struct {
	Decision *service.AssistantRewardDecision `json:"decision"`
	Granted  bool                             `json:"granted"`
}

func (h *AssistantHandler) Status(c *gin.Context) { response.Success(c, h.assistant.Status()) }

func (h *AssistantHandler) RewardStatus(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "authentication required")
		return
	}
	status, err := h.assistant.RewardStatus(c.Request.Context(), subject.UserID)
	if h.writeError(c, err) {
		return
	}
	response.Success(c, status)
}

func (h *AssistantHandler) ApplyReward(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "authentication required")
		return
	}
	decision, granted, err := h.assistant.ApplyNewUserReward(c.Request.Context(), subject.UserID)
	if h.writeError(c, err) {
		return
	}
	h.recordReward(c, subject.UserID, decision, granted)
	response.Success(c, &assistantRewardApplyResponse{Decision: decision, Granted: granted})
}

func (h *AssistantHandler) ListRewards(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	result, err := h.assistant.ListRewardDecisions(c.Request.Context(), page, pageSize)
	if h.writeError(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *AssistantHandler) ChatUser(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok || subject.UserID <= 0 {
		response.Unauthorized(c, "authentication required")
		return
	}
	var req assistantChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "question is required")
		return
	}
	reply, err := h.assistant.ChatUser(c.Request.Context(), subject.UserID, req.Question, req.History)
	if h.writeError(c, err) {
		return
	}
	h.record(c, subject.UserID, "user")
	response.Success(c, reply)
}

func (h *AssistantHandler) ChatAdmin(c *gin.Context) {
	var req assistantChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "question is required")
		return
	}
	reply, err := h.assistant.ChatAdmin(c.Request.Context(), req.Question, req.History)
	if h.writeError(c, err) {
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	h.record(c, subject.UserID, "admin")
	response.Success(c, reply)
}

func (h *AssistantHandler) OperationsSummary(c *gin.Context) {
	v, e := h.operations.Summary(c.Request.Context())
	if h.writeOperationError(c, e) {
		return
	}
	response.Success(c, v)
}
func (h *AssistantHandler) ListOperationConnections(c *gin.Context) {
	v, e := h.operations.ListConnections(c.Request.Context())
	if h.writeOperationError(c, e) {
		return
	}
	response.Success(c, v)
}
func (h *AssistantHandler) SaveOperationConnection(c *gin.Context) {
	var in service.OperationConnectionInput
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, "invalid connection")
		return
	}
	v, e := h.operations.SaveConnection(c.Request.Context(), 0, in)
	if h.writeOperationError(c, e) {
		return
	}
	response.Success(c, v)
}
func (h *AssistantHandler) UpdateOperationConnection(c *gin.Context) {
	id, ok := operationID(c)
	if !ok {
		return
	}
	var in service.OperationConnectionInput
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, "invalid connection")
		return
	}
	v, e := h.operations.SaveConnection(c.Request.Context(), id, in)
	if h.writeOperationError(c, e) {
		return
	}
	response.Success(c, v)
}
func (h *AssistantHandler) DeleteOperationConnection(c *gin.Context) {
	id, ok := operationID(c)
	if !ok {
		return
	}
	if h.writeOperationError(c, h.operations.DeleteConnection(c.Request.Context(), id)) {
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
func (h *AssistantHandler) TestOperationConnection(c *gin.Context) {
	id, ok := operationID(c)
	if !ok {
		return
	}
	if h.writeOperationError(c, h.operations.TestConnection(c.Request.Context(), id)) {
		return
	}
	response.Success(c, gin.H{"ok": true})
}
func (h *AssistantHandler) GetOperationPolicy(c *gin.Context) {
	v, e := h.operations.GetPolicy(c.Request.Context())
	if h.writeOperationError(c, e) {
		return
	}
	response.Success(c, v)
}
func (h *AssistantHandler) UpdateOperationPolicy(c *gin.Context) {
	var in service.OperationPolicy
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, "invalid policy")
		return
	}
	v, e := h.operations.UpdatePolicy(c.Request.Context(), in)
	if h.writeOperationError(c, e) {
		return
	}
	response.Success(c, v)
}

type operationTaskRequest struct {
	ConnectionID int64      `json:"connection_id" binding:"required"`
	Kind         string     `json:"kind"`
	Content      string     `json:"content" binding:"required"`
	ScheduledAt  *time.Time `json:"scheduled_at"`
}

func (h *AssistantHandler) CreateOperationTask(c *gin.Context) {
	var in operationTaskRequest
	if c.ShouldBindJSON(&in) != nil {
		response.BadRequest(c, "invalid task")
		return
	}
	scheduled := time.Now().UTC()
	if in.ScheduledAt != nil {
		scheduled = *in.ScheduledAt
	}
	if in.Kind == "" {
		in.Kind = "promotion"
	}
	v, e := h.operations.CreateTask(c.Request.Context(), in.ConnectionID, in.Kind, in.Content, scheduled)
	if h.writeOperationError(c, e) {
		return
	}
	response.Success(c, v)
}
func (h *AssistantHandler) ListOperationTasks(c *gin.Context) {
	page, size := response.ParsePagination(c)
	v, total, e := h.operations.ListTasks(c.Request.Context(), page, size)
	if h.writeOperationError(c, e) {
		return
	}
	response.Paginated(c, v, total, page, size)
}
func (h *AssistantHandler) ListOperationRuns(c *gin.Context) {
	page, size := response.ParsePagination(c)
	v, total, e := h.operations.ListRuns(c.Request.Context(), page, size)
	if h.writeOperationError(c, e) {
		return
	}
	response.Paginated(c, v, total, page, size)
}
func (h *AssistantHandler) ApproveOperationTask(c *gin.Context) {
	id, ok := operationID(c)
	if !ok {
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	v, e := h.operations.ApproveTask(c.Request.Context(), id, subject.UserID)
	if h.writeOperationError(c, e) {
		return
	}
	response.Success(c, v)
}
func (h *AssistantHandler) CancelOperationTask(c *gin.Context) {
	id, ok := operationID(c)
	if !ok {
		return
	}
	if h.writeOperationError(c, h.operations.CancelTask(c.Request.Context(), id)) {
		return
	}
	response.Success(c, gin.H{"cancelled": true})
}
func (h *AssistantHandler) RunOperationTask(c *gin.Context) {
	id, ok := operationID(c)
	if !ok {
		return
	}
	if h.writeOperationError(c, h.operations.RunTask(c.Request.Context(), id)) {
		return
	}
	response.Success(c, gin.H{"executed": true})
}
func (h *AssistantHandler) PlanOperationsNow(c *gin.Context) {
	if h.writeOperationError(c, h.operations.PlanNow(c.Request.Context())) {
		return
	}
	response.Success(c, gin.H{"planned": true})
}
func operationID(c *gin.Context) (int64, bool) {
	id, e := strconv.ParseInt(c.Param("id"), 10, 64)
	if e != nil || id <= 0 {
		response.BadRequest(c, "invalid id")
		return 0, false
	}
	return id, true
}
func (h *AssistantHandler) writeOperationError(c *gin.Context, e error) bool {
	if e == nil {
		return false
	}
	switch {
	case errors.Is(e, service.ErrOperationNotFound):
		response.NotFound(c, "operation resource not found")
	case errors.Is(e, service.ErrOperationInvalid):
		response.BadRequest(c, "invalid operation request")
	case errors.Is(e, service.ErrOperationPolicyBlocked):
		response.Error(c, http.StatusConflict, e.Error())
	default:
		response.Error(c, http.StatusBadGateway, "operation failed")
	}
	return true
}

func (h *AssistantHandler) writeError(c *gin.Context, err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, service.ErrAssistantDisabled):
		response.Error(c, http.StatusServiceUnavailable, "assistant is not configured")
	case errors.Is(err, service.ErrAssistantInvalidQuestion):
		response.BadRequest(c, "question is empty or too long")
	case errors.Is(err, service.ErrAssistantInvalidHistory):
		response.BadRequest(c, "conversation history is invalid")
	case errors.Is(err, service.ErrAssistantRewardDisabled):
		response.Error(c, http.StatusServiceUnavailable, "assistant reward is not enabled")
	default:
		response.Error(c, http.StatusBadGateway, "assistant is temporarily unavailable")
	}
	return true
}

func (h *AssistantHandler) recordReward(c *gin.Context, userID int64, decision *service.AssistantRewardDecision, granted bool) {
	extra := map[string]any{"granted": granted}
	if decision != nil {
		extra["decision_id"] = decision.ID
		extra["decision_status"] = decision.Status
		extra["final_amount"] = decision.FinalAmount
	}
	h.audit.Record(&service.AuditLog{
		CreatedAt:   time.Now().UTC(),
		ActorUserID: &userID,
		ActorRole:   "user",
		Action:      service.AuditActionAssistantRewardApply,
		Method:      c.Request.Method,
		Path:        c.FullPath(),
		RequestID:   c.GetHeader("X-Request-ID"),
		ClientIP:    c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		StatusCode:  http.StatusOK,
		Extra:       extra,
	})
}

func (h *AssistantHandler) record(c *gin.Context, userID int64, role string) {
	var actor *int64
	if userID > 0 {
		actor = &userID
	}
	h.audit.Record(&service.AuditLog{
		CreatedAt:   time.Now().UTC(),
		ActorUserID: actor,
		ActorRole:   role,
		Action:      service.AuditActionAssistantChat,
		Method:      c.Request.Method,
		Path:        c.FullPath(),
		RequestID:   c.GetHeader("X-Request-ID"),
		ClientIP:    c.ClientIP(),
		UserAgent:   c.Request.UserAgent(),
		StatusCode:  http.StatusOK,
	})
}
