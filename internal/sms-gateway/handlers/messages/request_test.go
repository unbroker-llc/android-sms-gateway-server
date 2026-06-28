package messages

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/android-sms-gateway/server/internal/sms-gateway/handlers/base"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap/zaptest"
)

// TestThirdPartySendRequest_Validate drives the real BodyParserValidator path
// (JSON parse -> validator.Var(&req, "required,dive") -> req.Validate()) so it
// proves three things at once:
//   - the embedded smsgateway.Message field tags still recurse (phoneNumbers
//     stays required) through struct embedding,
//   - our Validate() shadows the embedded Message.Validate() (which would reject
//     mmsMessage), so an MMS-only body is accepted,
//   - mutually-exclusive content + attachment rules are enforced.
//
// A nil error from the route yields 200; any validation error yields 500
// (fiber's default error handler), matching base/handler_test.go.
func TestThirdPartySendRequest_Validate(t *testing.T) {
	handler := &base.Handler{
		Logger:    zaptest.NewLogger(t),
		Validator: validator.New(),
	}

	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		var req thirdPartySendRequest
		return handler.BodyParserValidator(c, &req)
	})

	tests := []struct {
		name           string
		body           string
		expectedStatus int
	}{
		{
			name:           "MMS only with attachment is accepted",
			body:           `{"phoneNumbers":["+15551234567"],"mmsMessage":{"text":"hi","attachments":[{"contentType":"image/jpeg","name":"a.jpg","data":"SGVsbG8gV29ybGQh"}]}}`,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "MMS text-only (no attachments) is accepted",
			body:           `{"phoneNumbers":["+15551234567"],"mmsMessage":{"text":"just text"}}`,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Text only is still accepted",
			body:           `{"phoneNumbers":["+15551234567"],"textMessage":{"text":"hello"}}`,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Data only is still accepted",
			body:           `{"phoneNumbers":["+15551234567"],"dataMessage":{"data":"SGVsbG8=","port":53739}}`,
			expectedStatus: fiber.StatusOK,
		},
		{
			name:           "Empty content is rejected",
			body:           `{"phoneNumbers":["+15551234567"]}`,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "Missing phoneNumbers is rejected",
			body:           `{"mmsMessage":{"text":"hi","attachments":[{"contentType":"image/jpeg","data":"SGVsbG8="}]}}`,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "Conflicting text + mms is rejected",
			body:           `{"phoneNumbers":["+15551234567"],"textMessage":{"text":"hi"},"mmsMessage":{"text":"hi"}}`,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "MMS attachment missing contentType is rejected",
			body:           `{"phoneNumbers":["+15551234567"],"mmsMessage":{"attachments":[{"data":"SGVsbG8="}]}}`,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "MMS attachment with invalid base64 is rejected",
			body:           `{"phoneNumbers":["+15551234567"],"mmsMessage":{"attachments":[{"contentType":"image/jpeg","data":"!!!not base64!!!"}]}}`,
			expectedStatus: fiber.StatusInternalServerError,
		},
		{
			name:           "MMS with empty subject/text/attachments is rejected",
			body:           `{"phoneNumbers":["+15551234567"],"mmsMessage":{}}`,
			expectedStatus: fiber.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tc.body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test failed: %v", err)
			}
			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tc.expectedStatus)
			}
		})
	}
}
