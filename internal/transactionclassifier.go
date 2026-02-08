package internal

type TransactionClassifier struct {
	rules map[string]string
}

func BuildTransactionClassifier() *TransactionClassifier {
	return &TransactionClassifier{
		rules: map[string]string{
			"Leisure:Netflix": `name contains "NETFLIX" and amount < 0`,
			"Leisure:Amazon":  `name contains "AMAZON" and amount < 0`,
		},
	}
}

func (tc *TransactionClassifier) Classify() error {
	return nil
}
