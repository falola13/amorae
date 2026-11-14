package questions

// SetBank swaps the question list, so a test can use a small one and reach
// "every question has been used" without answering a hundred days.
func (s *Service) SetBank(b []string) { s.bank = b }
