package llm

import (
	"context"
	"fmt"
)

func CompleteStream(ctx context.Context, providers []Provider, req CompletionRequest) error {
	var lastErr error
	for _, p := range providers {
		ch, err := p.Stream(ctx, req)
		if err != nil {
			if isHTTPError(err) {
				lastErr = err
				continue
			}
			return fmt.Errorf("provider %s: %w", p.Name(), err)
		}
		for token := range ch {
			if token.Error != nil {
				return fmt.Errorf("provider %s: %s", p.Name(), token.Error)
			}
			if token.Done {
				return nil
			}
			fmt.Print(token.Text)
		}
		return nil
	}
	if lastErr != nil {
		return fmt.Errorf("all providers exhausted, last error: %w", lastErr)
	}
	return fmt.Errorf("no providers configured")
}
