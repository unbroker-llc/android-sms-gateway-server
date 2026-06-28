package messages

import (
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
)

type TextMessageContent = smsgateway.TextMessage
type DataMessageContent = smsgateway.DataMessage
type HashedMessageContent = smsgateway.HashedMessage

// MmsAttachment is a single MMS part. Mirrors the shape the Android app's
// cloud-pull parser expects (me.capcom.smsgateway.modules.gateway.GatewayApi
// MessageContent.Mms.Attachment): contentType, optional name, base64 data.
// The device reads bytes from `data` only - it has no URL fetch field - so
// outbound attachments must carry inline base64.
type MmsAttachment struct {
	ContentType string  `json:"contentType"`    // e.g. image/jpeg
	Name        *string `json:"name,omitempty"` // suggested filename
	Data        string  `json:"data"`           // base64-encoded bytes
}

// MmsMessageContent is the outbound MMS payload. Serialized to JSON as the
// `mmsMessage` object on both the 3rdparty send request and the device
// cloud-pull response.
type MmsMessageContent struct {
	Subject     *string         `json:"subject,omitempty"`
	Text        *string         `json:"text,omitempty"`
	Attachments []MmsAttachment `json:"attachments,omitempty"`
}

type MessageContent struct {
	TextContent *TextMessageContent `json:"textContent,omitempty"`
	DataContent *DataMessageContent `json:"dataContent,omitempty"`
	MmsContent  *MmsMessageContent  `json:"mmsContent,omitempty"`
}

type MessageStateContent struct {
	MessageContent

	HashedContent *HashedMessageContent `json:"hashedContent,omitempty"`
}

type MessageInput struct {
	MessageContent

	ID string

	PhoneNumbers []string
	IsEncrypted  bool

	SimNumber          *uint8
	WithDeliveryReport *bool
	TTL                *uint64
	ValidUntil         *time.Time
	ScheduleAt         *time.Time
	Priority           smsgateway.MessagePriority
}

type Message struct {
	MessageInput

	CreatedAt time.Time
}

type MessageStateInput struct {
	ID         string                      `json:"id"`         // Message ID
	State      ProcessingState             `json:"state"`      // State
	Recipients []smsgateway.RecipientState `json:"recipients"` // Recipients states
	States     map[string]time.Time        `json:"states"`     // History of states
}

type MessageState struct {
	MessageStateInput
	MessageStateContent

	DeviceID    string `json:"deviceId"`    // Device ID
	IsHashed    bool   `json:"isHashed"`    // Hashed
	IsEncrypted bool   `json:"isEncrypted"` // Encrypted
}
