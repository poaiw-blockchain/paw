package integration

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// TestAccount represents a test account with keys and balances
type TestAccount struct {
	Name       string
	PrivKey    cryptotypes.PrivKey
	PubKey     cryptotypes.PubKey
	Address    sdk.AccAddress
	Mnemonic   string
	AccountNum uint64
	Sequence   uint64
}

// NewTestAccount creates a new test account
func NewTestAccount(name string, accountNum uint64) *TestAccount {
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	return &TestAccount{
		Name:       name,
		PrivKey:    privKey,
		PubKey:     pubKey,
		Address:    addr,
		AccountNum: accountNum,
		Sequence:   0,
	}
}

// NewTestAccountWithKey creates a test account from an existing private key
func NewTestAccountWithKey(name string, privKey cryptotypes.PrivKey, accountNum uint64) *TestAccount {
	pubKey := privKey.PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	return &TestAccount{
		Name:       name,
		PrivKey:    privKey,
		PubKey:     pubKey,
		Address:    addr,
		AccountNum: accountNum,
		Sequence:   0,
	}
}

// String returns a string representation of the account
func (a *TestAccount) String() string {
	return fmt.Sprintf(
		"TestAccount{Name: %s, Address: %s, AccountNum: %d, Sequence: %d}",
		a.Name,
		a.Address.String(),
		a.AccountNum,
		a.Sequence,
	)
}

// IncrementSequence increments the account sequence
func (a *TestAccount) IncrementSequence() {
	a.Sequence++
}

// ToBaseAccount converts to Cosmos SDK BaseAccount
func (a *TestAccount) ToBaseAccount() *authtypes.BaseAccount {
	return authtypes.NewBaseAccount(a.Address, a.PubKey, a.AccountNum, a.Sequence)
}

// TestAccountManager manages multiple test accounts
type TestAccountManager struct {
	accounts map[string]*TestAccount
	counter  uint64
}

// NewTestAccountManager creates a new account manager
func NewTestAccountManager() *TestAccountManager {
	return &TestAccountManager{
		accounts: make(map[string]*TestAccount),
		counter:  0,
	}
}

// CreateAccount creates a new test account
func (m *TestAccountManager) CreateAccount(name string) *TestAccount {
	acc := NewTestAccount(name, m.counter)
	m.accounts[name] = acc
	m.counter++
	return acc
}

// CreateAccountWithKey creates account from existing key
func (m *TestAccountManager) CreateAccountWithKey(name string, privKey cryptotypes.PrivKey) *TestAccount {
	acc := NewTestAccountWithKey(name, privKey, m.counter)
	m.accounts[name] = acc
	m.counter++
	return acc
}

// GetAccount retrieves an account by name
func (m *TestAccountManager) GetAccount(name string) (*TestAccount, error) {
	acc, ok := m.accounts[name]
	if !ok {
		return nil, fmt.Errorf("account %s not found", name)
	}
	return acc, nil
}

// GetAllAccounts returns all accounts
func (m *TestAccountManager) GetAllAccounts() []*TestAccount {
	accounts := make([]*TestAccount, 0, len(m.accounts))
	for _, acc := range m.accounts {
		accounts = append(accounts, acc)
	}
	return accounts
}

// CreateFundedAccounts creates multiple pre-funded accounts
func (m *TestAccountManager) CreateFundedAccounts(count int, prefix string, funding sdk.Coins) ([]*TestAccount, []banktypes.Balance) {
	accounts := make([]*TestAccount, count)
	balances := make([]banktypes.Balance, count)

	for i := 0; i < count; i++ {
		name := fmt.Sprintf("%s-%d", prefix, i)
		acc := m.CreateAccount(name)
		accounts[i] = acc

		balances[i] = banktypes.Balance{
			Address: acc.Address.String(),
			Coins:   funding,
		}
	}

	return accounts, balances
}

// AccountSet represents a pre-configured set of test accounts
type AccountSet struct {
	Alice      *TestAccount
	Bob        *TestAccount
	Charlie    *TestAccount
	Dave       *TestAccount
	Validator1 *TestAccount
	Validator2 *TestAccount
	Validator3 *TestAccount
	Module     *TestAccount
}

// CreateDefaultAccountSet creates a default set of test accounts
func CreateDefaultAccountSet() *AccountSet {
	manager := NewTestAccountManager()

	return &AccountSet{
		Alice:      manager.CreateAccount("alice"),
		Bob:        manager.CreateAccount("bob"),
		Charlie:    manager.CreateAccount("charlie"),
		Dave:       manager.CreateAccount("dave"),
		Validator1: manager.CreateAccount("validator1"),
		Validator2: manager.CreateAccount("validator2"),
		Validator3: manager.CreateAccount("validator3"),
		Module:     manager.CreateAccount("module"),
	}
}

// GetAll returns all accounts in the set as a slice
func (s *AccountSet) GetAll() []*TestAccount {
	return []*TestAccount{
		s.Alice,
		s.Bob,
		s.Charlie,
		s.Dave,
		s.Validator1,
		s.Validator2,
		s.Validator3,
		s.Module,
	}
}

// GetValidators returns validator accounts
func (s *AccountSet) GetValidators() []*TestAccount {
	return []*TestAccount{
		s.Validator1,
		s.Validator2,
		s.Validator3,
	}
}

// GetUsers returns user accounts (non-validators)
func (s *AccountSet) GetUsers() []*TestAccount {
	return []*TestAccount{
		s.Alice,
		s.Bob,
		s.Charlie,
		s.Dave,
	}
}

// FundAccount is a helper to fund an account
type FundAccount struct {
	Account *TestAccount
	Amount  sdk.Coins
}

// NewFundAccount creates a funding instruction
func NewFundAccount(account *TestAccount, amount sdk.Coins) FundAccount {
	return FundAccount{
		Account: account,
		Amount:  amount,
	}
}

// CreateGenesisAccounts creates genesis accounts from test accounts
func CreateGenesisAccounts(accounts []*TestAccount) []authtypes.GenesisAccount {
	genesisAccounts := make([]authtypes.GenesisAccount, len(accounts))
	for i, acc := range accounts {
		genesisAccounts[i] = acc.ToBaseAccount()
	}
	return genesisAccounts
}

// CreateGenesisBalances creates genesis balances from test accounts
func CreateGenesisBalances(accounts []*TestAccount, funding sdk.Coins) []banktypes.Balance {
	balances := make([]banktypes.Balance, len(accounts))
	for i, acc := range accounts {
		balances[i] = banktypes.Balance{
			Address: acc.Address.String(),
			Coins:   funding,
		}
	}
	return balances
}
