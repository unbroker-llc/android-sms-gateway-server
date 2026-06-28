package converters_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/android-sms-gateway/server/internal/sms-gateway/handlers/converters"
	"github.com/android-sms-gateway/server/internal/sms-gateway/modules/messages"
	"github.com/go-playground/assert/v2"
	"github.com/samber/lo"
)

func TestMessageToDTO(t *testing.T) {
	// Set up a fixed time for testing
	now := time.Now().UTC()

	// Define test cases
	tests := []struct {
		name     string
		input    messages.Message
		expected converters.MobileMessageDTO
	}{
		{
			name: "Full message with all fields",
			input: messages.Message{
				MessageInput: messages.MessageInput{
					MessageContent: messages.MessageContent{
						TextContent: &messages.TextMessageContent{Text: "Test message content"},
					},

					ID:                 "msg-123",
					PhoneNumbers:       []string{"+1234567890", "+9876543210"},
					IsEncrypted:        true,
					SimNumber:          lo.ToPtr(uint8(2)),
					WithDeliveryReport: lo.ToPtr(true),
					TTL:                lo.ToPtr(uint64(3600)),
					ValidUntil:         lo.ToPtr(now.Add(24 * time.Hour)),
					Priority:           100,
				},
				CreatedAt: now,
			},
			expected: converters.MobileMessageDTO{
				MobileMessage: smsgateway.MobileMessage{
					Message: smsgateway.Message{
						ID:                 "msg-123",
						Message:            "Test message content",
						TextMessage:        &smsgateway.TextMessage{Text: "Test message content"},
						PhoneNumbers:       []string{"+1234567890", "+9876543210"},
						IsEncrypted:        true,
						SimNumber:          lo.ToPtr(uint8(2)),
						WithDeliveryReport: lo.ToPtr(true),
						TTL:                lo.ToPtr(uint64(3600)),
						ValidUntil:         lo.ToPtr(now.Add(24 * time.Hour)),
						Priority:           100,
					},
					CreatedAt: now,
				},
			},
		},
		{
			name: "Minimal message with required fields only",
			input: messages.Message{
				MessageInput: messages.MessageInput{
					MessageContent: messages.MessageContent{
						TextContent: &messages.TextMessageContent{Text: "Another test message"},
					},

					ID:           "msg-456",
					PhoneNumbers: []string{"+1122334455"},
				},
				CreatedAt: now,
			},
			expected: converters.MobileMessageDTO{
				MobileMessage: smsgateway.MobileMessage{
					Message: smsgateway.Message{
						ID:           "msg-456",
						Message:      "Another test message",
						TextMessage:  &smsgateway.TextMessage{Text: "Another test message"},
						PhoneNumbers: []string{"+1122334455"},
					},
					CreatedAt: now,
				},
			},
		},
	}

	// Execute tests
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Call the function under test
			result := converters.MessageToMobileDTO(tc.input)

			// Assert the results
			assert.Equal(t, tc.expected, result)
		})
	}
}

// TestMessageToMobileDTO_MMS_JSON is the load-bearing check: the device pulls
// these bytes and parses them with GatewayApi.Message. The MMS payload MUST
// serialize as a top-level `mmsMessage` object (not nested under another key)
// with `subject`, `text`, and `attachments[].{contentType,name,data}` - the
// exact shape the app's cloud-pull parser reads.
func TestMessageToMobileDTO_MMS_JSON(t *testing.T) {
	now := time.Now().UTC()

	input := messages.Message{
		MessageInput: messages.MessageInput{
			MessageContent: messages.MessageContent{
				MmsContent: &messages.MmsMessageContent{
					Subject: lo.ToPtr("Hello"),
					Text:    lo.ToPtr("See attached"),
					Attachments: []messages.MmsAttachment{
						{
							ContentType: "image/jpeg",
							Name:        lo.ToPtr("photo.jpg"),
							Data:        "SGVsbG8gV29ybGQh",
						},
					},
				},
			},
			ID:           "mms-1",
			PhoneNumbers: []string{"+15551234567"},
		},
		CreatedAt: now,
	}

	dto := converters.MessageToMobileDTO(input)

	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	js := string(raw)

	// Round-trip into a generic map to assert top-level structure precisely.
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	mms, ok := parsed["mmsMessage"].(map[string]any)
	if !ok {
		t.Fatalf("expected top-level `mmsMessage` object, got: %s", js)
	}
	if mms["subject"] != "Hello" {
		t.Errorf("mmsMessage.subject = %v, want Hello", mms["subject"])
	}
	if mms["text"] != "See attached" {
		t.Errorf("mmsMessage.text = %v, want 'See attached'", mms["text"])
	}

	atts, ok := mms["attachments"].([]any)
	if !ok || len(atts) != 1 {
		t.Fatalf("expected 1 attachment, got: %v", mms["attachments"])
	}
	att := atts[0].(map[string]any)
	if att["contentType"] != "image/jpeg" {
		t.Errorf("attachment.contentType = %v, want image/jpeg", att["contentType"])
	}
	if att["name"] != "photo.jpg" {
		t.Errorf("attachment.name = %v, want photo.jpg", att["name"])
	}
	if att["data"] != "SGVsbG8gV29ybGQh" {
		t.Errorf("attachment.data = %v, want base64 payload", att["data"])
	}

	// Sibling top-level fields (from the embedded MobileMessage) must still be
	// present and flat (phoneNumbers, id, createdAt), and text/data content keys
	// must be absent for an MMS-only message.
	if _, ok := parsed["phoneNumbers"].([]any); !ok {
		t.Errorf("expected top-level phoneNumbers array, got: %s", js)
	}
	if parsed["id"] != "mms-1" {
		t.Errorf("expected top-level id mms-1, got: %v", parsed["id"])
	}
	if _, exists := parsed["textMessage"]; exists {
		t.Errorf("textMessage should be omitted for MMS-only message: %s", js)
	}
	if _, exists := parsed["dataMessage"]; exists {
		t.Errorf("dataMessage should be omitted for MMS-only message: %s", js)
	}
	// The deprecated `message` string should be empty/omitted too.
	if v, exists := parsed["message"]; exists && v != "" {
		t.Errorf("deprecated `message` should be empty for MMS-only message, got: %v", v)
	}
	_ = strings.TrimSpace(js)
}
