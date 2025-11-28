package main

/*
PAW Blockchain - Create Wallet Example

This example demonstrates how to create a new wallet with a mnemonic phrase.

Usage:
    go run create_wallet.go              # Create new wallet
    go run create_wallet.go import "<mnemonic>"  # Import existing wallet

Security Warning:
    Never share your mnemonic phrase or private key with anyone.
    Store them securely offline.
*/

import (
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
)

const (
	// Wallet configuration
	WalletPrefix = "paw"
	HDPath       = "m/44'/118'/0'/0/0"
	CoinType     = 118
	Account      = 0
	Index        = 0
)

// Wallet represents a PAW blockchain wallet
type Wallet struct {
	Mnemonic   string
	PrivateKey *secp256k1.PrivKey
	PublicKey  []byte
	Address    string
}

// GenerateMnemonic generates a new 24-word mnemonic
func GenerateMnemonic() (string, error) {
	// Generate 256 bits of entropy (24 words)
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", fmt.Errorf("failed to generate entropy: %w", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	return mnemonic, nil
}

// CreateWallet creates a new wallet from a mnemonic
func CreateWallet(mnemonic string) (*Wallet, error) {
	// Validate mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		return nil, fmt.Errorf("invalid mnemonic phrase")
	}

	// Derive seed from mnemonic
	seed := bip39.NewSeed(mnemonic, "")

	// Derive master private key
	master, ch := hd.ComputeMastersFromSeed(seed)

	// Derive key using BIP44 path
	derivedPriv, err := hd.DerivePrivateKeyForPath(master, ch, HDPath)
	if err != nil {
		return nil, fmt.Errorf("failed to derive private key: %w", err)
	}

	// Create private key
	privKey := &secp256k1.PrivKey{Key: derivedPriv}

	// Get public key
	pubKey := privKey.PubKey().Bytes()

	// Generate address
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(WalletPrefix, WalletPrefix+"pub")
	addr := sdk.AccAddress(privKey.PubKey().Address()).String()

	return &Wallet{
		Mnemonic:   mnemonic,
		PrivateKey: privKey,
		PublicKey:  pubKey,
		Address:    addr,
	}, nil
}

// PrintWallet prints wallet information
func PrintWallet(wallet *Wallet, isNew bool) {
	fmt.Println(strings.Repeat("=", 80))
	if isNew {
		fmt.Println("WALLET CREATED SUCCESSFULLY")
	} else {
		fmt.Println("WALLET IMPORTED SUCCESSFULLY")
	}
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("\n⚠️  SECURITY WARNING: Keep this information secure and private!\n")

	if isNew {
		fmt.Println("Mnemonic Phrase (24 words):")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println(wallet.Mnemonic)
		fmt.Println(strings.Repeat("-", 80))
	}

	fmt.Println("\nWallet Details:")
	fmt.Printf("  Address:    %s\n", wallet.Address)
	fmt.Printf("  Public Key: %X\n", wallet.PublicKey)
	fmt.Printf("  HD Path:    %s\n", HDPath)
	fmt.Printf("  Prefix:     %s\n", WalletPrefix)

	if isNew {
		fmt.Println("\nNext Steps:")
		fmt.Println("  1. Save your mnemonic phrase in a secure location")
		fmt.Println("  2. Never share your mnemonic with anyone")
		fmt.Println("  3. Fund your wallet with PAW tokens")
		fmt.Println("  4. Use your address to receive tokens")
	}
}

func main() {
	args := os.Args[1:]

	if len(args) > 0 && args[0] == "import" {
		// Import existing wallet
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "Usage: go run create_wallet.go import \"<mnemonic>\"")
			os.Exit(1)
		}

		mnemonic := strings.Join(args[1:], " ")
		fmt.Println("Importing Existing Wallet...\n")

		wallet, err := CreateWallet(mnemonic)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Error importing wallet: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✓ Wallet imported successfully")
		fmt.Printf("\nAddress: %s\n\n", wallet.Address)

	} else {
		// Create new wallet
		fmt.Println("Creating New PAW Wallet...\n")

		mnemonic, err := GenerateMnemonic()
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Error generating mnemonic: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✓ Generated new mnemonic phrase")

		wallet, err := CreateWallet(mnemonic)
		if err != nil {
			fmt.Fprintf(os.Stderr, "✗ Error creating wallet: %v\n", err)
			os.Exit(1)
		}

		PrintWallet(wallet, true)
	}
}
