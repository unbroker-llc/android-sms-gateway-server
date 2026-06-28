package messages

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/android-sms-gateway/server/internal/sms-gateway/modules/messages"
)

// thirdPartySendRequest is the body of POST /3rdparty/v1/messages.
//
// It embeds the upstream smsgateway.Message (so all existing fields - id,
// deviceId, phoneNumbers, textMessage, dataMessage, simNumber, ttl, etc. - are
// promoted and parsed identically) and adds the outbound MMS content field.
// The added MmsMessage field is serialized/parsed as a top-level `mmsMessage`
// JSON object, matching docs/MMS.md and the device's cloud-pull parser.
//
// A local Validate() shadows the embedded smsgateway.Message.Validate() (which
// only permits text/data content); this one additionally accepts mmsMessage.
type thirdPartySendRequest struct {
	smsgateway.Message

	MmsMessage *messages.MmsMessageContent `json:"mmsMessage,omitempty"`
}

// Validate enforces "exactly one content type" across text / data / mms (plus
// the deprecated `message` string), mirrors the upstream ttl/validUntil and
// scheduleAt checks, and validates MMS attachments.
func (r *thirdPartySendRequest) Validate() error {
	contentCount := 0
	if r.Message.Message != "" {
		contentCount++
	}
	if r.Message.TextMessage != nil {
		contentCount++
	}
	if r.Message.DataMessage != nil {
		contentCount++
	}
	if r.MmsMessage != nil {
		contentCount++
	}

	if contentCount != 1 {
		return errors.New("must specify exactly one of: textMessage, dataMessage or mmsMessage")
	}

	if r.Message.TTL != nil && r.Message.ValidUntil != nil {
		return errors.New("conflicting fields: ttl and validUntil")
	}

	if r.Message.ScheduleAt != nil && !r.Message.ScheduleAt.After(time.Now()) {
		return errors.New("scheduleAt must be in the future")
	}

	if r.MmsMessage != nil {
		return validateMms(r.MmsMessage)
	}

	return nil
}

func validateMms(m *messages.MmsMessageContent) error {
	hasText := m.Text != nil && strings.TrimSpace(*m.Text) != ""
	if !hasText && len(m.Attachments) == 0 {
		return errors.New("mmsMessage must have text or at least one attachment")
	}

	for i, att := range m.Attachments {
		if strings.TrimSpace(att.ContentType) == "" {
			return fmt.Errorf("attachment %d: contentType is required", i)
		}
		if strings.TrimSpace(att.Data) == "" {
			return fmt.Errorf("attachment %d: data is required", i)
		}
		if !isValidBase64(att.Data) {
			return fmt.Errorf("attachment %d: data must be valid base64", i)
		}
	}

	return nil
}

// isValidBase64 accepts standard base64, with or without padding (the device
// decodes with android.util.Base64.DEFAULT, i.e. standard alphabet). Surrounding
// whitespace/newlines are tolerated.
func isValidBase64(s string) bool {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', ' ', '\t':
			return -1
		}
		return r
	}, s)

	if _, err := base64.StdEncoding.DecodeString(cleaned); err == nil {
		return true
	}
	if _, err := base64.RawStdEncoding.DecodeString(cleaned); err == nil {
		return true
	}

	return false
}
