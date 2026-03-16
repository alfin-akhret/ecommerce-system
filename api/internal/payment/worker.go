package payment

/**
func (s *Service) ExpirePayment(ctx context.Context) error {
	payments, err := s.repo.FindExpiredPayment(ctx)
	if err != nil {
		return err
	}

	for _, payment := range payments {
		tx, err := s.db.Begin(ctx)
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)

		paymentRepo := s.repo.WithTx(tx)

		payment.Status = "EXPIRED"
		err = paymentRepo.UpdateStatus(ctx, payment.ID.String(), payment.Status, nil)
		if err != nil {
			return err
		}


	}
}
*/
