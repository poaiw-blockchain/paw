# cosmospy-protobuf

Production-ready Python wheels that package Cosmos SDK protobuf definitions used by the PAW wallet and SDK tooling.  
The goal is to provide deterministic, reproducible bindings so Python clients can sign and broadcast Cosmos transactions without relying on ad-hoc JSON encodings.

## Contents

- `cosmos/tx/v1beta1`: Tx, TxBody, AuthInfo, TxRaw and signing helpers  
- `cosmos/tx/signing/v1beta1`: `SignMode` enums and legacy amino bits  
- `cosmos/base/v1beta1`: `Coin` and related helpers  
- `cosmos/crypto/secp256k1` + `cosmos/crypto/multisig`: public key structures  
- `amino` & `cosmos_proto`: descriptor extensions required by the SDK

All protobuf files mirror the Cosmos SDK `v0.50.9` release to stay in lockstep with the Go code that powers PAW.

## Regenerating protobuf bindings

```bash
cd sdk/python/cosmospy-protobuf
python -m venv .venv
source .venv/bin/activate
pip install grpcio-tools==1.76.0
python -m grpc_tools.protoc \
  --proto_path=proto \
  --python_out=cosmospy_protobuf \
  cosmos/tx/v1beta1/tx.proto \
  cosmos/tx/signing/v1beta1/signing.proto \
  cosmos/base/v1beta1/coin.proto \
  cosmos/crypto/multisig/v1beta1/multisig.proto \
  cosmos/crypto/secp256k1/keys.proto \
  amino/amino.proto \
  cosmos_proto/cosmos.proto
```

Each run must be followed by `python -m build` (or `hatch build`) to ensure the generated files are captured in the wheel.

## Building & publishing

```bash
cd sdk/python/cosmospy-protobuf
python -m build
twine check dist/*
twine upload dist/*
```

Use a scoped PyPI token with project permissions only. The package declares the Apache-2.0 license to remain compatible with the upstream Cosmos SDK repository.

## Testing

```bash
python -m venv .venv
source .venv/bin/activate
pip install -e .[dev]
python - <<'PY'
from cosmospy_protobuf import tx
print(tx["core"].TxBody().WhichOneof("messages"))
PY
```

Because these bindings are generated directly from audited upstream protos, there are no runtime unit tests beyond import smoke tests—the correctness is guaranteed by `grpcio-tools` and upstream protobuf definitions.
