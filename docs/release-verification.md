# Verifying releases

Every release ships with a `checksums.txt` file, which has the SHA-256 sum of each available archive.
This file is signed with [cosign](https://github.com/sigstore/cosign) using keyless [Sigstore](https://www.sigstore.dev/) signing.
The file is accompanied by the signature, `checksums.txt.sig`, and the signing certificate, `checksums.txt.pem`.

The install script already checks the checksum of what it downloads.
The steps below go one step further and prove that `checksums.txt` itself was produced by this repository's release workflow.
In other words, it proves it wasn't tampered with or re-signed by someone else.

To follow along, you need to have [`cosign`](https://github.com/sigstore/cosign) installed.

```sh
VERSION=v0.7.0   # the release you're verifying
base="https://github.com/cleura/cleura-cli/releases/download/$VERSION"

# 1. Fetch the checksum file, its signature, and the signing certificate.
curl -fsSLO "$base/checksums.txt"
curl -fsSLO "$base/checksums.txt.sig"
curl -fsSLO "$base/checksums.txt.pem"

# 2. Verify checksums.txt was signed by this repo's release workflow.
cosign verify-blob checksums.txt \
  --signature checksums.txt.sig \
  --certificate checksums.txt.pem \
  --certificate-identity "https://github.com/cleura/cleura-cli/.github/workflows/release.yml@refs/tags/$VERSION" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com"

# 3. checksums.txt is now trusted — verify your downloaded archive against it
# (run from the directory where you downloaded the archive).
sha256sum --ignore-missing -c checksums.txt # macOS: shasum -a 256 --ignore-missing -c checksums.txt
```

On success, `cosign verify-blob` returns `Verified OK`.
To verify any release without editing the tag, swap `--certificate-identity` for `--certificate-identity-regexp '^https://github\.com/cleura/cleura-cli/\.github/workflows/release\.yml@refs/tags/v'`.
