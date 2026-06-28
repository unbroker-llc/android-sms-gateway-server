-- +goose Up
-- +goose StatementBegin
ALTER TABLE `messages`
MODIFY `type` enum('Text', 'Data', 'Mms') NOT NULL DEFAULT 'Text';
-- +goose StatementEnd
---
-- +goose Down
-- +goose StatementBegin
ALTER TABLE `messages`
MODIFY `type` enum('Text', 'Data') NOT NULL DEFAULT 'Text';
-- +goose StatementEnd
