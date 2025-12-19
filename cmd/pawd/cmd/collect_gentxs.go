package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	tmtypes "github.com/cometbft/cometbft/types"
)

// CollectGenTxsCmd returns a command to collect genesis transactions
func CollectGenTxsCmd(mbm module.BasicManager, defaultNodeHome string, genBalIterator genutiltypes.GenesisBalancesIterator, validator genutiltypes.MessageValidator) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "collect-gentxs",
		Short: "Collect genesis txs and output a genesis.json file",
		Long: `Collect genesis transactions from the configured gentx directory and
update the genesis file with the collected transactions.

Example:
  pawd collect-gentxs --home ~/.paw
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			serverCtx := server.GetServerContextFromCmd(cmd)
			clientCtx := client.GetClientContextFromCmd(cmd)

			config := serverCtx.Config
			config.SetRoot(clientCtx.HomeDir)

			// Read genesis file
			genFile := config.GenesisFile()
			// Normalize numeric fields if they are encoded as strings (common in canonicalized genesis)
			if err := normalizeGenesisNumbers(genFile); err != nil {
				return fmt.Errorf("failed to normalize genesis: %w", err)
			}

			genDoc, err := tmtypes.GenesisDocFromFile(genFile)
			if err != nil {
				return fmt.Errorf("failed to read genesis doc from file %s: %w", genFile, err)
			}

			// Unmarshal genesis state
			var genesisState map[string]json.RawMessage
			if err = json.Unmarshal(genDoc.AppState, &genesisState); err != nil {
				return fmt.Errorf("failed to unmarshal genesis state: %w", err)
			}

			// Read gentx files
			gentxDir := filepath.Join(config.RootDir, "config", "gentx")
			gentxFiles, err := os.ReadDir(gentxDir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(cmd.OutOrStdout(), "No gentx directory found at %s; leaving genesis unchanged.\n", gentxDir)
					return nil
				}
				return fmt.Errorf("failed to read gentx directory: %w", err)
			}

			if len(gentxFiles) == 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "No gentx files found in %s; leaving genesis unchanged.\n", gentxDir)
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Collecting genesis transactions...\n")
			fmt.Fprintf(cmd.OutOrStdout(), "Found %d gentx files\n", len(gentxFiles))

			// Collect all gentxs
			var (
				genTxs            []sdk.Tx
				genesisValidators []tmtypes.GenesisValidator
				msgCreateVals     []*stakingtypes.MsgCreateValidator
				seenValidators    = make(map[string]struct{})
			)
			for _, gentxFile := range gentxFiles {
				if gentxFile.IsDir() || filepath.Ext(gentxFile.Name()) != ".json" {
					continue
				}

				gentxPath := filepath.Join(gentxDir, gentxFile.Name())

				// Read gentx file
				gentxBz, err := os.ReadFile(gentxPath) // #nosec G304 - gentx files are operator supplied
				if err != nil {
					return fmt.Errorf("failed to read gentx file %s: %w", gentxPath, err)
				}

				// Decode gentx
				tx, err := clientCtx.TxConfig.TxJSONDecoder()(gentxBz)
				if err != nil {
					return fmt.Errorf("failed to decode gentx %s: %w", gentxPath, err)
				}

				// Validate gentx
				msgs := tx.GetMsgs()
				if len(msgs) != 1 {
					return fmt.Errorf("gentx must contain exactly one message, got %d", len(msgs))
				}

				// Verify it's a MsgCreateValidator
				msgCreateVal, ok := msgs[0].(*stakingtypes.MsgCreateValidator)
				if !ok {
					return fmt.Errorf("gentx message must be MsgCreateValidator")
				}

				// ValidateBasic was removed in SDK v0.50 - validation happens in message server
				// Basic validation: check that required fields are present
				if msgCreateVal.ValidatorAddress == "" {
					return fmt.Errorf("invalid gentx: validator address is required")
				}
				if msgCreateVal.Pubkey == nil {
					return fmt.Errorf("invalid gentx: pubkey is required")
				}

				if _, exists := seenValidators[msgCreateVal.ValidatorAddress]; exists {
					return fmt.Errorf("duplicate gentx for validator %s", msgCreateVal.ValidatorAddress)
				}
				seenValidators[msgCreateVal.ValidatorAddress] = struct{}{}

				validator, err := msgCreateValidatorToGenesisValidator(clientCtx.InterfaceRegistry, msgCreateVal)
				if err != nil {
					return err
				}

				genesisValidators = append(genesisValidators, validator)
				genTxs = append(genTxs, tx)
				msgCreateVals = append(msgCreateVals, msgCreateVal)
				fmt.Fprintf(cmd.OutOrStdout(), "  ✓ Collected gentx from %s\n", gentxFile.Name())
			}

			// Update genesis state with collected gentxs
			genUtilGenesis := genutiltypes.GetGenesisStateFromAppState(clientCtx.Codec, genesisState)
			bankGenesis := banktypes.GetGenesisStateFromAppState(clientCtx.Codec, genesisState)
			stakingGenesis := stakingtypes.GetGenesisStateFromAppState(clientCtx.Codec, genesisState)

			// Get slashing genesis to add signing info for each validator
			var slashingGenesis slashingtypes.GenesisState
			if genesisState[slashingtypes.ModuleName] != nil {
				clientCtx.Codec.MustUnmarshalJSON(genesisState[slashingtypes.ModuleName], &slashingGenesis)
			} else {
				slashingGenesis = *slashingtypes.DefaultGenesisState()
			}

			bondedPoolAddress := authtypes.NewModuleAddress(stakingtypes.BondedPoolName).String()
			bondedPoolBalance := ensureBalance(&bankGenesis.Balances, bondedPoolAddress)

			stakingGenesis.Validators = make([]stakingtypes.Validator, 0, len(msgCreateVals))
			stakingGenesis.Delegations = make([]stakingtypes.Delegation, 0, len(msgCreateVals))
			stakingGenesis.LastValidatorPowers = make([]stakingtypes.LastValidatorPower, 0, len(msgCreateVals))

			lastTotalPower := math.NewInt(0)
			bondDenom := stakingGenesis.Params.BondDenom

			for idx, msg := range msgCreateVals {
				if msg.Value.Denom != bondDenom {
					return fmt.Errorf("gentx %d uses %s but bond denom is %s", idx+1, msg.Value.Denom, bondDenom)
				}

				delegatorAddr := msg.DelegatorAddress
				if delegatorAddr == "" {
					valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
					if err != nil {
						return fmt.Errorf("invalid validator address %s: %w", msg.ValidatorAddress, err)
					}
					delegatorAddr = sdk.AccAddress(valAddr).String()
				}

				delegatorBalance := findBalance(bankGenesis.Balances, delegatorAddr)
				if delegatorBalance == nil {
					return fmt.Errorf("delegator %s has no balance entry in genesis", delegatorAddr)
				}

				if delegatorBalance.Coins.AmountOf(msg.Value.Denom).LT(msg.Value.Amount) {
					return fmt.Errorf("delegator %s insufficient balance for self-delegation", delegatorAddr)
				}

				delegatorBalance.Coins = delegatorBalance.Coins.Sub(msg.Value)
				bondedPoolBalance.Coins = bondedPoolBalance.Coins.Add(msg.Value)

				shares := math.LegacyNewDecFromInt(msg.Value.Amount)
				validator := stakingtypes.Validator{
					OperatorAddress:         msg.ValidatorAddress,
					ConsensusPubkey:         msg.Pubkey,
					Jailed:                  false,
					Status:                  stakingtypes.Bonded,
					Tokens:                  msg.Value.Amount,
					DelegatorShares:         shares,
					Description:             msg.Description,
					UnbondingHeight:         0,
					UnbondingTime:           time.Unix(0, 0).UTC(),
					Commission:              stakingtypes.Commission{CommissionRates: msg.Commission, UpdateTime: time.Unix(0, 0).UTC()},
					MinSelfDelegation:       msg.MinSelfDelegation,
					UnbondingOnHoldRefCount: 0,
				}

				stakingGenesis.Validators = append(stakingGenesis.Validators, validator)
				stakingGenesis.Delegations = append(stakingGenesis.Delegations, stakingtypes.Delegation{
					DelegatorAddress: delegatorAddr,
					ValidatorAddress: msg.ValidatorAddress,
					Shares:           shares,
				})

				power := sdk.TokensToConsensusPower(msg.Value.Amount, sdk.DefaultPowerReduction)
				stakingGenesis.LastValidatorPowers = append(stakingGenesis.LastValidatorPowers, stakingtypes.LastValidatorPower{
					Address: msg.ValidatorAddress,
					Power:   power,
				})
				lastTotalPower = lastTotalPower.Add(math.NewInt(power))

				// Create slashing signing info for this validator
				// This is required for the slashing module to track validator uptime
				var pubKey cryptotypes.PubKey
				if err := clientCtx.InterfaceRegistry.UnpackAny(msg.Pubkey, &pubKey); err != nil {
					return fmt.Errorf("failed to unpack validator pubkey for slashing info: %w", err)
				}
				consAddr := sdk.ConsAddress(pubKey.Address())
				signingInfo := slashingtypes.NewValidatorSigningInfo(
					consAddr,
					0,                       // StartHeight - will be set at first block
					0,                       // IndexOffset
					time.Unix(0, 0).UTC(),   // JailedUntil
					false,                   // Tombstoned
					0,                       // MissedBlocksCounter
				)
				slashingGenesis.SigningInfos = append(slashingGenesis.SigningInfos, slashingtypes.SigningInfo{
					Address:              consAddr.String(),
					ValidatorSigningInfo: signingInfo,
				})
			}

			stakingGenesis.LastTotalPower = lastTotalPower
			genUtilGenesis.GenTxs = []json.RawMessage{}

			genesisState[banktypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(bankGenesis)
			genesisState[stakingtypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(stakingGenesis)
			genesisState[slashingtypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(&slashingGenesis)
			genesisState[genutiltypes.ModuleName] = clientCtx.Codec.MustMarshalJSON(genUtilGenesis)

			// Validate genesis state
			if err = mbm.ValidateGenesis(clientCtx.Codec, clientCtx.TxConfig, genesisState); err != nil {
				return fmt.Errorf("failed to validate genesis state: %w", err)
			}

			// Marshal updated genesis state
			appStateJSON, err := json.MarshalIndent(genesisState, "", " ")
			if err != nil {
				return fmt.Errorf("failed to marshal genesis state: %w", err)
			}

			genDoc.AppState = appStateJSON
			genDoc.Validators = genesisValidators

			// Validate and complete genesis doc
			if err = genDoc.ValidateAndComplete(); err != nil {
				return fmt.Errorf("failed to validate genesis doc: %w", err)
			}

			// Save updated genesis file
			if err = genDoc.SaveAs(genFile); err != nil {
				return fmt.Errorf("failed to save genesis file: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\nSuccessfully collected %d genesis transactions\n", len(genTxs))
			fmt.Fprintf(cmd.OutOrStdout(), "Genesis file updated: %s\n", genFile)
			fmt.Fprintf(cmd.OutOrStdout(), "\nValidators:\n")
			for i, tx := range genTxs {
				msgCreateVal := tx.GetMsgs()[0].(*stakingtypes.MsgCreateValidator)
				fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s (%s)\n", i+1, msgCreateVal.Description.Moniker, msgCreateVal.ValidatorAddress)
			}

			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")
	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

func msgCreateValidatorToGenesisValidator(registry codectypes.InterfaceRegistry, msg *stakingtypes.MsgCreateValidator) (tmtypes.GenesisValidator, error) {
	if msg == nil {
		return tmtypes.GenesisValidator{}, fmt.Errorf("msg create validator cannot be nil")
	}

	var pubKey cryptotypes.PubKey
	if err := registry.UnpackAny(msg.Pubkey, &pubKey); err != nil {
		return tmtypes.GenesisValidator{}, fmt.Errorf("failed to unpack validator pubkey: %w", err)
	}

	consensusPubKey, err := cryptocodec.ToCmtPubKeyInterface(pubKey)
	if err != nil {
		return tmtypes.GenesisValidator{}, fmt.Errorf("failed to convert validator pubkey: %w", err)
	}

	power := sdk.TokensToConsensusPower(msg.Value.Amount, sdk.DefaultPowerReduction)
	if power <= 0 {
		return tmtypes.GenesisValidator{}, fmt.Errorf("validator %s has zero consensus power", msg.ValidatorAddress)
	}

	return tmtypes.GenesisValidator{
		Address: consensusPubKey.Address(),
		PubKey:  consensusPubKey,
		Power:   power,
		Name:    msg.Description.Moniker,
	}, nil
}

func findBalance(balances []banktypes.Balance, address string) *banktypes.Balance {
	for i := range balances {
		if balances[i].Address == address {
			return &balances[i]
		}
	}
	return nil
}

func ensureBalance(balances *[]banktypes.Balance, address string) *banktypes.Balance {
	for i := range *balances {
		if (*balances)[i].Address == address {
			return &(*balances)[i]
		}
	}

	*balances = append(*balances, banktypes.Balance{
		Address: address,
		Coins:   sdk.NewCoins(),
	})

	return &(*balances)[len(*balances)-1]
}
