package cmd

import (
	"bufio"
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"

	"github.com/paw-chain/paw/app"
)

const (
	flagMnemonicLength = "mnemonic-length"
	flagNoBackup       = "no-backup"
	flagKeyType        = "key-type"
	flagCoinType       = "coin-type"
	flagAccount        = "account"
	flagIndex          = "index"
	// flagRecover is defined in init.go
)

// KeysCmd returns the keys command with PAW-specific BIP39 enhancements.
// It includes the --home flag when invoked standalone. When wired under the
// root command, use newKeysCmd(false) to avoid duplicate persistent flag
// definitions.
func KeysCmd() *cobra.Command {
	return newKeysCmd(true)
}

func newKeysCmd(includeHomeFlag bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage your application's keys with BIP39 mnemonic support",
		Long: `Keys allows you to manage your local keystore for tendermint.

    These keys may be in any format supported by the Tendermint keyring library.

    PAW enhances the standard keys command with comprehensive BIP39 mnemonic support:
    - Generate 12-word or 24-word mnemonics
    - Recover keys from existing mnemonics
    - Secure entropy generation using crypto/rand
    - Automatic mnemonic validation with checksums`,
	}

	// Provide home/keyring flags when keys is invoked standalone (without pawd root).
	if includeHomeFlag && cmd.PersistentFlags().Lookup(flags.FlagHome) == nil {
		cmd.PersistentFlags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")
	}
	if cmd.PersistentFlags().Lookup(flags.FlagKeyringBackend) == nil {
		cmd.PersistentFlags().String(flags.FlagKeyringBackend, flags.DefaultKeyringBackend, "Select keyring backend (os|file|kwallet|pass|test|memory)")
	}

	cmd.AddCommand(
		AddKeyCommand(),
		RecoverKeyCommand(),
		ListKeysCommand(),
		ShowKeysCommand(),
		DeleteKeyCommand(),
		ExportKeyCommand(),
		ImportKeyCommand(),
	)

	return cmd
}

// AddKeyCommand creates a new key in the keyring with mnemonic generation
func AddKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new key with BIP39 mnemonic generation",
		Long: `Add a new encrypted key to the keyring with BIP39 mnemonic phrase.

The key will be generated using secure random entropy and a BIP39 mnemonic phrase
will be displayed for backup purposes. You can choose between 12-word (128-bit) or
24-word (256-bit) mnemonics.

WARNING: Keep your mnemonic phrase in a secure location. Anyone with access to your
mnemonic can recover your private keys and access your funds.

Examples:
  pawd keys add mykey                           # Generate 24-word mnemonic (default)
  pawd keys add mykey --mnemonic-length 12      # Generate 12-word mnemonic
  pawd keys add mykey --no-backup               # Skip mnemonic display (not recommended)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := getKeyringClientContext(cmd)
			if err != nil {
				return err
			}

			name := strings.TrimSpace(args[0])
			if name == "" {
				return fmt.Errorf("argument 'name' cannot be empty")
			}

			if recoverExisting, _ := cmd.Flags().GetBool("recover"); recoverExisting {
				return recoverKeyWithPrompt(cmd, clientCtx, name)
			}

			// Get flags
			mnemonicLength, _ := cmd.Flags().GetInt(flagMnemonicLength)
			noBackup, _ := cmd.Flags().GetBool(flagNoBackup)
			keyType, _ := cmd.Flags().GetString(flagKeyType)
			coinType, _ := cmd.Flags().GetUint32(flagCoinType)
			account, _ := cmd.Flags().GetUint32(flagAccount)
			index, _ := cmd.Flags().GetUint32(flagIndex)

			// Validate mnemonic length
			if mnemonicLength != 12 && mnemonicLength != 24 {
				return fmt.Errorf("mnemonic length must be 12 or 24 words")
			}

			// Generate entropy based on mnemonic length
			// 12 words = 128 bits, 24 words = 256 bits
			var entropySize int
			if mnemonicLength == 12 {
				entropySize = 128 / 8 // 16 bytes
			} else {
				entropySize = 256 / 8 // 32 bytes
			}

			// Generate secure entropy using crypto/rand
			entropy := make([]byte, entropySize)
			if _, err := rand.Read(entropy); err != nil {
				return fmt.Errorf("failed to generate secure entropy: %w", err)
			}

			// Generate mnemonic from entropy
			mnemonic, err := bip39.NewMnemonic(entropy)
			if err != nil {
				return fmt.Errorf("failed to generate mnemonic: %w", err)
			}

			// Validate the generated mnemonic (sanity check)
			if !bip39.IsMnemonicValid(mnemonic) {
				return fmt.Errorf("generated mnemonic failed validation")
			}

			// Create key from mnemonic
			hdPath := hd.CreateHDPath(coinType, account, index)
			key, err := clientCtx.Keyring.NewAccount(
				name,
				mnemonic,
				keyring.DefaultBIP39Passphrase,
				hdPath.String(),
				hd.Secp256k1,
			)
			if err != nil {
				return fmt.Errorf("failed to create key: %w", err)
			}

			// Get address
			addr, err := key.GetAddress()
			if err != nil {
				return fmt.Errorf("failed to get address: %w", err)
			}

			// Display key information
			fmt.Fprintf(cmd.OutOrStdout(), "\n")
			fmt.Fprintf(cmd.OutOrStdout(), "- name: %s\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "  type: %s\n", keyType)
			fmt.Fprintf(cmd.OutOrStdout(), "  address: %s\n", addr.String())
			fmt.Fprintf(cmd.OutOrStdout(), "  pubkey: %s\n", key.PubKey.String())
			fmt.Fprintf(cmd.OutOrStdout(), "\n")

			// Display mnemonic (unless --no-backup is set)
			if !noBackup {
				fmt.Fprintf(cmd.OutOrStdout(), "**IMPORTANT** Write this mnemonic phrase in a safe place.\n")
				fmt.Fprintf(cmd.OutOrStdout(), "It is the only way to recover your account if you ever forget your password.\n")
				fmt.Fprintf(cmd.OutOrStdout(), "\n")
				fmt.Fprintf(cmd.OutOrStdout(), "%s\n", mnemonic)
				fmt.Fprintf(cmd.OutOrStdout(), "\n")
			}

			return nil
		},
	}

	cmd.Flags().Bool("recover", false, "Recover key from an existing mnemonic instead of generating a new one")
	cmd.Flags().Int(flagMnemonicLength, 24, "Mnemonic length (12 or 24 words)")
	cmd.Flags().Bool(flagNoBackup, false, "Skip mnemonic backup prompt (WARNING: not recommended)")
	cmd.Flags().String(flagKeyType, "secp256k1", "Key signing algorithm")
	cmd.Flags().Uint32(flagCoinType, sdk.GetConfig().GetCoinType(), "Coin type number for HD derivation")
	cmd.Flags().Uint32(flagAccount, 0, "Account number for HD derivation")
	cmd.Flags().Uint32(flagIndex, 0, "Address index number for HD derivation")
	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendOS, "Keyring backend")
	cmd.Flags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")

	return cmd
}

// recoverKeyWithPrompt recovers a key using interactive mnemonic entry.
func recoverKeyWithPrompt(cmd *cobra.Command, clientCtx client.Context, name string) error {
	// Get flags
	coinType, _ := cmd.Flags().GetUint32(flagCoinType)
	account, _ := cmd.Flags().GetUint32(flagAccount)
	index, _ := cmd.Flags().GetUint32(flagIndex)

	// Prompt for mnemonic
	buf := bufio.NewReader(cmd.InOrStdin())
	mnemonic, err := input.GetString("Enter your bip39 mnemonic", buf)
	if err != nil {
		return fmt.Errorf("failed to read mnemonic: %w", err)
	}

	// Clean up mnemonic (remove extra spaces, normalize)
	mnemonic = strings.TrimSpace(mnemonic)
	words := strings.Fields(mnemonic)
	mnemonic = strings.Join(words, " ")

	// Validate mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		return fmt.Errorf("invalid mnemonic: checksum failed")
	}

	// Count words
	wordCount := len(words)
	if wordCount != 12 && wordCount != 24 {
		return fmt.Errorf("invalid mnemonic length: expected 12 or 24 words, got %d", wordCount)
	}

	// Validate entropy size matches word count
	expectedEntropyBits := (wordCount * 11) - (wordCount / 3)
	if (wordCount == 12 && expectedEntropyBits != 128) || (wordCount == 24 && expectedEntropyBits != 256) {
		return fmt.Errorf("mnemonic word count doesn't match expected entropy")
	}

	// Create key from mnemonic
	hdPath := hd.CreateHDPath(coinType, account, index)
	key, err := clientCtx.Keyring.NewAccount(
		name,
		mnemonic,
		keyring.DefaultBIP39Passphrase,
		hdPath.String(),
		hd.Secp256k1,
	)
	if err != nil {
		return fmt.Errorf("failed to recover key: %w", err)
	}

	addr, err := key.GetAddress()
	if err != nil {
		return fmt.Errorf("failed to get address: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n")
	fmt.Fprintf(cmd.OutOrStdout(), "- name: %s\n", name)
	fmt.Fprintf(cmd.OutOrStdout(), "  type: local\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  address: %s\n", addr.String())
	fmt.Fprintf(cmd.OutOrStdout(), "  pubkey: %s\n", key.PubKey.String())
	fmt.Fprintf(cmd.OutOrStdout(), "\n")

	fmt.Fprintf(cmd.OutOrStdout(), "Key successfully recovered from %d-word mnemonic!\n", wordCount)
	return nil
}

// RecoverKeyCommand recovers a key from a BIP39 mnemonic
func RecoverKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "recover [name]",
		Short: "Recover a key from BIP39 mnemonic phrase",
		Long: `Recover a key from an existing BIP39 mnemonic phrase.

The mnemonic will be validated for proper BIP39 checksum before being used to
generate the key. This command supports both 12-word and 24-word mnemonics.

Examples:
  pawd keys recover mykey
  pawd keys recover mykey --account 1 --index 0`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := getKeyringClientContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]

			// Get flags
			coinType, _ := cmd.Flags().GetUint32(flagCoinType)
			account, _ := cmd.Flags().GetUint32(flagAccount)
			index, _ := cmd.Flags().GetUint32(flagIndex)

			// Prompt for mnemonic
			buf := bufio.NewReader(cmd.InOrStdin())
			mnemonic, err := input.GetString("Enter your bip39 mnemonic", buf)
			if err != nil {
				return fmt.Errorf("failed to read mnemonic: %w", err)
			}

			// Clean up mnemonic (remove extra spaces, normalize)
			mnemonic = strings.TrimSpace(mnemonic)
			words := strings.Fields(mnemonic)
			mnemonic = strings.Join(words, " ")

			// Validate mnemonic
			if !bip39.IsMnemonicValid(mnemonic) {
				return fmt.Errorf("invalid mnemonic: checksum failed")
			}

			// Count words
			wordCount := len(words)
			if wordCount != 12 && wordCount != 24 {
				return fmt.Errorf("invalid mnemonic length: expected 12 or 24 words, got %d", wordCount)
			}

			// Validate entropy size matches word count
			// This is an additional sanity check
			expectedEntropyBits := (wordCount * 11) - (wordCount / 3)
			if (wordCount == 12 && expectedEntropyBits != 128) || (wordCount == 24 && expectedEntropyBits != 256) {
				return fmt.Errorf("mnemonic word count doesn't match expected entropy")
			}

			// Create key from mnemonic
			hdPath := hd.CreateHDPath(coinType, account, index)
			key, err := clientCtx.Keyring.NewAccount(
				name,
				mnemonic,
				keyring.DefaultBIP39Passphrase,
				hdPath.String(),
				hd.Secp256k1,
			)
			if err != nil {
				return fmt.Errorf("failed to recover key: %w", err)
			}

			// Get address
			addr, err := key.GetAddress()
			if err != nil {
				return fmt.Errorf("failed to get address: %w", err)
			}

			// Display recovered key information
			fmt.Fprintf(cmd.OutOrStdout(), "\n")
			fmt.Fprintf(cmd.OutOrStdout(), "- name: %s\n", name)
			fmt.Fprintf(cmd.OutOrStdout(), "  type: local\n")
			fmt.Fprintf(cmd.OutOrStdout(), "  address: %s\n", addr.String())
			fmt.Fprintf(cmd.OutOrStdout(), "  pubkey: %s\n", key.PubKey.String())
			fmt.Fprintf(cmd.OutOrStdout(), "  mnemonic: \"\"\n")
			fmt.Fprintf(cmd.OutOrStdout(), "\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Key successfully recovered from %d-word mnemonic!\n", wordCount)

			return nil
		},
	}

	cmd.Flags().Uint32(flagCoinType, sdk.GetConfig().GetCoinType(), "Coin type number for HD derivation")
	cmd.Flags().Uint32(flagAccount, 0, "Account number for HD derivation")
	cmd.Flags().Uint32(flagIndex, 0, "Address index number for HD derivation")
	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendOS, "Keyring backend")
	cmd.Flags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")

	return cmd
}

// ListKeysCommand lists all keys in the keyring
func ListKeysCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all keys",
		Long:  "List all keys stored in the keyring",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := getKeyringClientContext(cmd)
			if err != nil {
				return err
			}

			keys, err := clientCtx.Keyring.List()
			if err != nil {
				return err
			}

			if len(keys) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No keys found.\n")
				return nil
			}

			for _, key := range keys {
				addr, err := key.GetAddress()
				if err != nil {
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "- name: %s\n", key.Name)
				fmt.Fprintf(cmd.OutOrStdout(), "  address: %s\n", addr.String())
				fmt.Fprintf(cmd.OutOrStdout(), "\n")
			}

			return nil
		},
	}

	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendOS, "Keyring backend")
	cmd.Flags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")

	return cmd
}

// ShowKeysCommand shows key information
func ShowKeysCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show [name]",
		Short: "Show key information",
		Long:  "Show detailed information for a specific key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := getKeyringClientContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]
			key, err := clientCtx.Keyring.Key(name)
			if err != nil {
				return fmt.Errorf("key not found: %w", err)
			}

			addr, err := key.GetAddress()
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "- name: %s\n", key.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "  address: %s\n", addr.String())
			fmt.Fprintf(cmd.OutOrStdout(), "  pubkey: %s\n", key.PubKey.String())

			return nil
		},
	}

	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendOS, "Keyring backend")
	cmd.Flags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")

	return cmd
}

// DeleteKeyCommand deletes a key from the keyring
func DeleteKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete [name]",
		Short: "Delete a key",
		Long:  "Delete a key from the keyring. WARNING: This operation is irreversible unless you have a backup of your mnemonic.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := getKeyringClientContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]

			skipPrompt, _ := cmd.Flags().GetBool("yes")
			if !skipPrompt {
				buf := bufio.NewReader(cmd.InOrStdin())
				confirmation, err := input.GetString(fmt.Sprintf("Are you sure you want to delete key '%s'? [y/N]", name), buf)
				if err != nil {
					return err
				}

				if confirmation != "y" && confirmation != "Y" {
					fmt.Fprintf(cmd.OutOrStdout(), "Deletion cancelled.\n")
					return nil
				}
			}

			err = clientCtx.Keyring.Delete(name)
			if err != nil {
				return fmt.Errorf("failed to delete key: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Key '%s' deleted successfully.\n", name)
			return nil
		},
	}

	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendOS, "Keyring backend")
	cmd.Flags().Bool("yes", false, "skip confirmation prompt")
	cmd.Flags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")

	return cmd
}

// ExportKeyCommand exports a key in ASCII-armored format
func ExportKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [name]",
		Short: "Export a key in ASCII-armored encrypted format",
		Long:  "Export a private key from the local keyring in ASCII-armored encrypted format.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := getKeyringClientContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]

			// Get passphrase for encryption
			buf := bufio.NewReader(cmd.InOrStdin())
			passphrase, err := input.GetPassword("Enter passphrase to encrypt the exported key:", buf)
			if err != nil {
				return err
			}

			armor, err := clientCtx.Keyring.ExportPrivKeyArmor(name, passphrase)
			if err != nil {
				return fmt.Errorf("failed to export key: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s\n", armor)
			return nil
		},
	}

	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendOS, "Keyring backend")
	cmd.Flags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")

	return cmd
}

// ImportKeyCommand imports a key from ASCII-armored format
func ImportKeyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import [name] [keyfile]",
		Short: "Import a key from ASCII-armored encrypted format",
		Long:  "Import a private key into the local keyring from an ASCII-armored encrypted format.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := getKeyringClientContext(cmd)
			if err != nil {
				return err
			}

			name := args[0]
			keyfile := args[1]

			// Read key file
			armor, err := os.ReadFile(keyfile) // #nosec G304 - key import path provided by operator
			if err != nil {
				return fmt.Errorf("failed to read key file: %w", err)
			}

			// Get passphrase for decryption
			buf := bufio.NewReader(cmd.InOrStdin())
			passphrase, err := input.GetPassword("Enter passphrase to decrypt the key:", buf)
			if err != nil {
				return err
			}

			err = clientCtx.Keyring.ImportPrivKey(name, string(armor), passphrase)
			if err != nil {
				return fmt.Errorf("failed to import key: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Key '%s' imported successfully.\n", name)
			return nil
		},
	}

	cmd.Flags().String(flags.FlagKeyringBackend, keyring.BackendOS, "Keyring backend")
	cmd.Flags().String(flags.FlagHome, app.DefaultNodeHome, "directory for config and data")

	return cmd
}

func getKeyringClientContext(cmd *cobra.Command) (client.Context, error) {
	clientCtx := client.GetClientContextFromCmd(cmd)
	existingKeyring := clientCtx.Keyring

	clientCtx, err := client.ReadPersistentCommandFlags(clientCtx, cmd.Flags())
	if err != nil {
		return clientCtx, err
	}

	if existingKeyring != nil {
		clientCtx = clientCtx.WithKeyring(existingKeyring)
	}

	return clientCtx, nil
}
