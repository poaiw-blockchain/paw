# Multi-Language Wallet Provider Examples

The TypeScript SDK under `wallet/core` powers the desktop, mobile, and extension wallets, but many integrators prefer native providers for Go- and Rust-based infrastructure. This guide shows how to derive PAW accounts, sign transactions, and talk to RPC or gRPC endpoints using those languages so service providers can embed PAW support without going through a JavaScript bridge.

All snippets assume the PAW chain is running with `chain-id=paw-mvp-1`, gRPC on `localhost:9090`, and RPC on `http://localhost:26657`. Adjust the endpoints per your deployment.

---

## Go Provider Example

This example uses Cosmos SDK libraries to derive a key, sign a bank transfer, and broadcast via gRPC. It mirrors what the JS SDK does internally but keeps everything inside a Go-based service.

```go
package main

import (
	"context"
	"encoding/hex"
	"log"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"google.golang.org/grpc"
)

const mnemonic = "enlist hip relief stomach skate base shallow young switch frequent cry park"

func main() {
	encoding := MakeEncodingConfig() // helper from app or simapp
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount("paw", "pawpub")

	// 1. Derive private key (m/44'/118'/0'/0/0)
	algo := hd.Secp256k1
	derived, err := algo.Derive()(mnemonic, "", hd.CreateHDPath(118, 0, 0).String())
	if err != nil {
		log.Fatal(err)
	}
	privKey := &secp256k1.PrivKey{Key: derived}

	// 2. Build message
	fromAddr := sdk.AccAddress(privKey.PubKey().Address()).String()
	toAddr := "paw1f3h7csh3fvhe72p2c5dlz93ens5w0m00cqpv4q"
	msg := &banktypes.MsgSend{
		FromAddress: fromAddr,
		ToAddress:   toAddr,
		Amount:      sdk.Coins{sdk.NewInt64Coin("upaw", 10_000)},
	}

	// 3. Prepare factory + signer info
	factory := tx.Factory{}.
		WithChainID("paw-mvp-1").
		WithTxConfig(encoding.TxConfig).
		WithGas(120000).
		WithFees("2500upaw")

	// Query account number/sequence via gRPC
	conn, err := grpc.Dial("localhost:9090", grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	authClient := banktypes.NewQueryClient(conn)

	// ... build auth query context, omitted for brevity ...

	txBuilder := encoding.TxConfig.NewTxBuilder()
	if err := txBuilder.SetMsgs(msg); err != nil {
		log.Fatal(err)
	}
	txBuilder.SetGasLimit(120000)
	txBuilder.SetFeeAmount(sdk.Coins{sdk.NewInt64Coin("upaw", 2500)})

	sig := signingtypes.SignatureV2{
		PubKey: privKey.PubKey(),
		Data: &signingtypes.SingleSignatureData{
			SignMode:  encoding.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil, // populated by SignWithPrivKey
		},
		Sequence: 0,
	}

	if err := tx.SignWithPrivKey(
		encoding.TxConfig.SignModeHandler().DefaultMode(),
		sig, txBuilder, privKey, factory.TxConfig().SignModeHandler().DefaultMode(), 0,
	); err != nil {
		log.Fatal(err)
	}

	// 4. Broadcast
	txBytes, err := encoding.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.BroadcastTxSync(context.Background(), conn, txBytes)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Tx hash: %s", hex.EncodeToString(res.TxHash))
}
```

**Highlights**

- Uses native Cosmos SDK key derivation to match the JS wallet derivation path.
- Works inside any Go microservice—no Node.js dependency.
- Use the same factory + signer logic for oracle, compute, or DEX module messages by swapping the `Msg` type.

---

## Rust Provider Example

Rust operators can rely on the [`cosmrs`](https://docs.rs/cosmrs/latest/cosmrs/) crate. The snippet below derives a key, signs a DEX liquidity add transaction, and posts it to the REST endpoint. Replace message payloads as needed.

```rust
use cosmrs::{
    bip32::Mnemonic,
    crypto::secp256k1::SigningKey,
    tx::{self, SignDoc, SignerInfo},
    AccountId, Coin,
};
use cosmwasm_std::Uint128;
use serde_json::json;

fn main() -> anyhow::Result<()> {
    let mnemonic = Mnemonic::new(
        "enlist hip relief stomach skate base shallow young switch frequent cry park",
        Default::default(),
    );
    let signer = SigningKey::derive_from_path(&mnemonic, "m/44'/118'/0'/0/0")?;
    let account = AccountId::new("paw", signer.public_key().account_id("paw")?.as_ref())?;

    // Build custom PAW DEX message (example)
    let msg = json!({
        "@type": "/paw.dex.MsgAddLiquidity",
        "sender": account.to_string(),
        "pool_id": "1",
        "amount_a": { "denom": "upaw", "amount": "500000" },
        "amount_b": { "denom": "usdc", "amount": "200000" }
    });

    let body = tx::Body::new(vec![msg.try_into()?], "rust-provider", 0);
    let signer_info = SignerInfo::single_direct(Some(signer.public_key()), 0);
    let auth_info = signer_info.auth_info(Uint128::new(150_000), Coin::new(3_000u64, "upaw"));

    let sign_doc = SignDoc::new(&body, &auth_info, "paw-mvp-1", AccountNumber::new(0))?;
    let tx_raw = sign_doc.sign(&signer)?;

    let client = reqwest::blocking::Client::new();
    let res = client
        .post("http://localhost:1317/cosmos/tx/v1beta1/txs")
        .json(&json!({ "tx_bytes": base64::encode(tx_raw.to_bytes()), "mode": "BROADCAST_MODE_SYNC" }))
        .send()?;

    println!("Broadcast status: {}", res.text()?);
    Ok(())
}
```

**Highlights**

- Uses the upstream `cosmrs` tooling, so maintainers inherit all protobuf updates.
- Example shows JSON-based DEX message; replace with Oracle or Compute module types as needed.
- Works in stateless workers thanks to serialized `tx_bytes`.

---

## Operational Tips

1. **Derivation Consistency**: Stick to `m/44'/118'/0'/0/0` unless your org has reason to shard accounts. This ensures compatibility with the official wallet exports.
2. **Chain Params**: Source chain-id and denomination metadata from `docs/PARAMETER_REFERENCE.md` to avoid hard-coded fees.
3. **Offline Signing**: Both snippets can run in offline or air-gapped environments. Persist the `tx_bytes` and broadcast later via a bastion host.
4. **Testing**: Use `scripts/test-cli-commands.sh` or the integration suite under `tests/integration` to validate new providers before exposing them to users.

Add new languages by following the same pattern: deterministic derivation, message creation, `SignDoc`, and broadcast. Cross-reference this guide whenever wallet integrators ask for Go or Rust examples so PAW no longer relies solely on Python samples.
