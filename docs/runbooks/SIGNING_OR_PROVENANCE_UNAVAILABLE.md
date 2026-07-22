# Signing or provenance unavailable

1. Stop the release. Do not publish an unsigned substitute, reuse another environment identity, store a private key in repository secrets, or promote by mutable tag.
2. Preserve source revision, workflow run, image name/digest, completed SBOM/scan digests, and the redacted signing/attestation error.
3. Determine whether GitHub OIDC, GHCR, the transparency service, permissions, workflow identity, or a pinned tool/action failed. Verify repository/ref and minimal permissions before retrying.
4. If an image was pushed but not completely signed and attested, mark that digest untrusted and do not promote it. A retry may complete the same digest only from the same reviewed source/workflow; otherwise rebuild and use the new digest.
5. After recovery, verify Cosign certificate identity and issuer plus GitHub attestation repository/subject digest. Exercise wrong-source and tampered-digest rejection.
6. Record the outage and revalidation evidence. Availability pressure never authorizes bypass.
