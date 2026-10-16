package service

import (
	"encoding/json"
	"strings"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

type jevContextTurn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

// Only client-supplied current user content is newly audited. Tool loops retain
// their existing skip boundary. A reminder marker is data, never authorization.
func extractJevModerationInput(protocol string, body []byte) ContentModerationInput {
	out := extractContentModerationInput(protocol, body, false)
	out.OriginalRunes = utf8.RuneCountInString(out.Text)
	out.Normalize()
	if !jevNeedsContext(out.Text) {
		return out
	}
	var turns gjson.Result
	switch protocol {
	case ContentModerationProtocolOpenAIChat, ContentModerationProtocolAnthropicMessages:
		turns = gjson.GetBytes(body, "messages")
	case ContentModerationProtocolOpenAIResponses:
		turns = gjson.GetBytes(body, "input")
	case ContentModerationProtocolGemini:
		turns = gjson.GetBytes(body, "contents")
	default:
		return out
	}
	items := turns.Array()
	if len(items) < 2 {
		return out
	}
	current := len(items) - 1
	if protocol == ContentModerationProtocolAnthropicMessages {
		for current >= 0 && items[current].Get("role").String() == "system" {
			current--
		}
	}
	// Keep at most two adjacent user/assistant turns. Do not walk through tool,
	// system, or developer messages or reintroduce an entire historical prompt.
	for i := current - 1; i >= 0 && len(out.Context) < 2; i-- {
		item := items[i]
		role := strings.ToLower(item.Get("role").String())
		if role == "model" {
			role = "assistant"
		}
		if role != "user" && role != "assistant" {
			break
		}
		var text, images []string
		if protocol == ContentModerationProtocolGemini {
			item.Get("parts").ForEach(func(_, p gjson.Result) bool {
				if s := p.Get("text").String(); s != "" {
					text = append(text, s)
				}
				return true
			})
		} else {
			moderationTextCollector{}.collectContentValue(item.Get("content"), &text, &images)
		}
		normalized := normalizeContentModerationText(strings.Join(text, "\n"))
		if normalized == "" {
			break
		}
		out.Context = append([]jevContextTurn{{Role: role, Text: trimRunes(normalized, 600)}}, out.Context...)
	}
	return out
}

func jevNeedsContext(text string) bool {
	if text == "" || utf8.RuneCountInString(text) > 160 {
		return false
	}
	value := strings.ToLower(text)
	for _, marker := range []string{"继续", "上面", "上文", "上述", "刚才", "那段", "按你的", "按这个", "照做", "它的"} {
		if strings.Contains(value, marker) {
			return true
		}
	}
	for _, marker := range []string{"continue", "go on", "do that", "do it", "the above", "previous answer", "that code", "same thing"} {
		if value == marker || strings.HasPrefix(value, marker+" ") || strings.HasPrefix(value, marker+".") {
			return true
		}
	}
	return false
}

func jevModerationPayload(content ContentModerationInput) any {
	if len(content.Context) == 0 {
		return content.ModerationInput()
	}
	// JSON encoding prevents client text from forging our role/context delimiters.
	encoded, _ := json.Marshal(struct {
		CurrentRequest string           `json:"current_request"`
		RecentContext  []jevContextTurn `json:"recent_context"`
	}{content.Text, content.Context})
	return string(encoded)
}
