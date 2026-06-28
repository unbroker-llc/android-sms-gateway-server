package converters

import (
	"github.com/android-sms-gateway/client-go/smsgateway"
	"github.com/android-sms-gateway/server/internal/sms-gateway/modules/messages"
)

// MobileMessageDTO is the device-pull message DTO. It extends the upstream
// smsgateway.MobileMessage with the outbound MMS content. The MmsMessage field
// is promoted to a top-level `mmsMessage` JSON object (struct embedding
// flattens MobileMessage -> Message), matching the shape the Android app's
// cloud-pull parser reads (GatewayApi.Message._mmsMessage).
type MobileMessageDTO struct {
	smsgateway.MobileMessage

	MmsMessage *messages.MmsMessageContent `json:"mmsMessage,omitempty"`
}

func MessageToMobileDTO(m messages.Message) MobileMessageDTO {
	var message string
	var textMessage *smsgateway.TextMessage
	var dataMessage *smsgateway.DataMessage
	var mmsMessage *messages.MmsMessageContent

	switch {
	case m.TextContent != nil:
		message = m.TextContent.Text
		textMessage = &smsgateway.TextMessage{
			Text: m.TextContent.Text,
		}
	case m.DataContent != nil:
		dataMessage = &smsgateway.DataMessage{
			Data: m.DataContent.Data,
			Port: m.DataContent.Port,
		}
	case m.MmsContent != nil:
		mmsMessage = m.MmsContent
	}

	return MobileMessageDTO{
		MobileMessage: smsgateway.MobileMessage{
			Message: smsgateway.Message{
				ID:       m.ID,
				DeviceID: "",

				Message:     message,
				TextMessage: textMessage,
				DataMessage: dataMessage,

				SimNumber:          m.SimNumber,
				WithDeliveryReport: m.WithDeliveryReport,
				IsEncrypted:        m.IsEncrypted,
				PhoneNumbers:       m.PhoneNumbers,
				TTL:                m.TTL,
				ValidUntil:         m.ValidUntil,
				ScheduleAt:         m.ScheduleAt,
				Priority:           m.Priority,
			},
			CreatedAt: m.CreatedAt,
		},
		MmsMessage: mmsMessage,
	}
}

func MessageStateToDTO(state messages.MessageState) smsgateway.MessageState {
	return smsgateway.MessageState{
		ID:          state.ID,
		DeviceID:    state.DeviceID,
		State:       smsgateway.ProcessingState(state.State),
		IsHashed:    state.IsHashed,
		IsEncrypted: state.IsEncrypted,
		Recipients:  state.Recipients,
		States:      state.States,

		TextMessage:   state.TextContent,
		DataMessage:   state.DataContent,
		HashedMessage: state.HashedContent,
	}
}
