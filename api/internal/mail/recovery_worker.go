package mail

import (
	"context"
	"time"
)

func (s *Service) RecoverInbox(ctx context.Context) error {
	return s.inboxRepo.RecoverProcessing(ctx, 2*time.Minute, consumerName)
}
