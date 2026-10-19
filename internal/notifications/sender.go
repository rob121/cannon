package notifications

import (
	"context"
	"fmt"
	"strings"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/types"
)

func sendShoutrr(ctx context.Context, rawURL, subject, message string) error {
	_ = ctx // shoutrrr API is not context-aware yet
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("shoutrrr url is required")
	}
	message = strings.TrimSpace(message)
	subject = strings.TrimSpace(subject)

	sender, err := shoutrrr.CreateSender(rawURL)
	if err != nil {
		return fmt.Errorf("invalid shoutrrr url: %w", err)
	}
	params := &types.Params{}
	if subject != "" {
		params.SetTitle(subject)
	}
	errs := sender.Send(message, params)
	for _, sendErr := range errs {
		if sendErr != nil {
			return sendErr
		}
	}
	return nil
}
