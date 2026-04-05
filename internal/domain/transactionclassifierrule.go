package domain

import "github.com/expr-lang/expr/vm"

type TransactionClassifierRule struct {
	ID      uint64
	Rule    string
	Account Account
	Program *vm.Program
}
