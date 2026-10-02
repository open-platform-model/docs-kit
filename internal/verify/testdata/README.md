# Signature fixtures

Real artifacts from the phase-1 spike (GHCR package `docs/spike`, run
37053754999), kept so the verifier's identity policy is tested offline
against a certificate GitHub Actions and Fulcio really issued:

- `spike-manifest.json`: the bundle manifest, digest
  `sha256:3891fc3d62ce77fd33f838f7b7da4a780b1dfff554a39c1998614a6f8bc1399d`.
- `spike-referrer.json`: the cosign v3 referrer manifest whose subject is
  that digest, digest
  `sha256:86f70d3b491bd7451bd143efbd7f9ea4bcf976b3464916f69a07b8f4e5982b0e`.
- `spike-bundle.sigstore.json`: the referrer's one layer, the Sigstore
  bundle. Its certificate names
  `.github/workflows/spike-sign.yml@refs/heads/spike/ghcr` as the signer,
  `https://github.com/open-platform-model/docs-kit` as the Source Repository
  URI and `refs/heads/spike/ghcr` as the Source Repository Ref.
- `public-good-trusted-root.json`: a snapshot of the Sigstore public-good
  trusted root that verifies it.

The files are byte for byte what the registry served; a changed byte breaks
the digests and the signature.
