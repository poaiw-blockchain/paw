# Release Artifacts Policy

This repository is source-only. Do not commit compiled binaries, build outputs, or secrets.

Release artifacts must be published separately (for example, via GitHub Releases) with
checksums and signatures.

Example:
```
sha256sum my-artifact-linux-amd64 > SHA256SUMS
gpg --detach-sign --armor SHA256SUMS
# or
cosign sign-blob --key cosign.key SHA256SUMS
```
