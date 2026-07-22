# Verify Atlas release provenance

1. Obtain the immutable `ghcr.io/...@sha256:...` reference from the release manifest, not from a tag.
2. Run Cosign verification with the expected `MichaelSeveen/atlas/.github/workflows/release.yml` certificate-identity pattern and `https://token.actions.githubusercontent.com` issuer.
3. Run `gh attestation verify oci://<name@digest> --repo MichaelSeveen/atlas` and inspect source repository, workflow, commit, subject name, and subject digest.
4. Retrieve the SPDX SBOM attestation and compare its subject digest with the image. Review scanner and license results for that same digest.
5. Prove fail-closed behavior with a changed digest and an unexpected repository identity; both must fail. Never test tampering against the promoted registry object itself.
6. Record verifier versions, time, source revision, exact digest, expected/observed results, and sanitized output. Verification establishes origin/integrity evidence, not overall software security.
