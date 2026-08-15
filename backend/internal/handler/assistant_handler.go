package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AssistantHandler struct {
	assistant *service.AssistantService
	audit     *service.AuditLogService
}

func NewAssistantHandler(assistant *service.AssistantService, audit *service.AuditLogService) *AssistantHandler {
	return &AssistantHandler{assistant: assistant, audit: audit}
}

type assistantChatRequest struct {
	Question string `json:"question" binding:"required"`
}

func (h *AssistantHandler) Status(c *gin.Context) { response.Success(c, h.assistant.Status()) }

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
	reply, err := h.assistant.ChatUser(c.Request.Context(), subject.UserID, req.Question)
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
	reply, err := h.assistant.ChatAdmin(c.Request.Context(), req.Question)
	if h.writeError(c, err) {
		return
	}
	subject, _ := middleware.GetAuthSubjectFromContext(c)
	h.record(c, subject.UserID, "admin")
	response.Success(c, reply)
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
	default:
		response.Error(c, http.StatusBadGateway, "assistant is temporarily unavailable")
	}
	return true
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
