package web

import (
	"strings"

	"github.com/syrm/maille/internal/domain"
)

type categoryOption struct {
	ID    uint64
	Label string
	Icon  string
}

func categoryOptions(accounts []domain.Account) []categoryOption {
	categoryAccounts := make([]domain.Account, 0, len(accounts))
	aliases := make(map[string]int)
	for _, account := range accounts {
		if account.Type == domain.AccountTypeExpenses || account.Type == domain.AccountTypeIncome {
			categoryAccounts = append(categoryAccounts, account)
			aliases[account.Alias]++
		}
	}

	options := make([]categoryOption, 0, len(categoryAccounts))
	for _, account := range categoryAccounts {
		label := account.Alias
		if aliases[account.Alias] > 1 {
			nameParts := strings.Split(account.Name, ":")
			label += " · " + nameParts[len(nameParts)-1]
		}
		options = append(options, categoryOption{ID: account.ID, Label: label, Icon: account.Icon})
	}
	return options
}
