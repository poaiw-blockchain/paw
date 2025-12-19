package database

// Stub implementations to keep staging builds functional. These return
// empty objects to allow the API to respond without backing data.

func (db *Database) GetDEXPool(poolID string) (*DEXPool, error) {
	return &DEXPool{PoolID: poolID}, nil
}

func (db *Database) GetBlockByHeight(height int64) (*Block, error) {
	return &Block{Height: height}, nil
}

func (db *Database) SearchBlocksByHash(query string, offset, limit int) ([]Block, int, error) {
	return []Block{}, 0, nil
}

func (db *Database) GetTransactionByHash(hash string) (*Transaction, error) {
	return &Transaction{Hash: hash}, nil
}

func (db *Database) SearchTransactionsByHash(query string, offset, limit int) ([]Transaction, int, error) {
	return []Transaction{}, 0, nil
}

func (db *Database) GetAccount(address string) (*Account, error) {
	return &Account{Address: address}, nil
}

func (db *Database) SearchAccountsByAddress(query string, offset, limit int) ([]Account, int, error) {
	return []Account{}, 0, nil
}
