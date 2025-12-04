# Publish cosmospy-protobuf

1. **Regenerate protobuf bindings (optional)**
   ```bash
   cd sdk/python/cosmospy-protobuf
   python -m venv .venv
   source .venv/bin/activate
   pip install --upgrade pip
   pip install -e .[dev]
   scripts/generate_protos.sh
   ```

2. **Run smoke test**
   ```bash
   python - <<'PY'
   from cosmospy_protobuf import cosmos_tx_v1beta1_tx_pb2
   print("TxBody fields:", [f.name for f in cosmos_tx_v1beta1_tx_pb2.TxBody.DESCRIPTOR.fields])
   PY
   ```

3. **Build artifacts**
   ```bash
   python -m build
   twine check dist/*
   ```

4. **Publish**
   ```bash
   export TWINE_USERNAME="__token__"
   export TWINE_PASSWORD="<pypi-token>"
   twine upload dist/*
   ```

5. **Verify in clean environment**
   ```bash
   python -m venv /tmp/test-cosmospy
   source /tmp/test-cosmospy/bin/activate
   pip install cosmospy-protobuf==1.0.0
   python -c "import cosmospy_protobuf; print('ok', cosmospy_protobuf)"
   ```

6. **Bump dependency in `archive/sdk/python/pyproject.toml` if needed**

Publishing from CI is recommended once the manual flow is validated. Upload tokens should be scoped to this single project.
