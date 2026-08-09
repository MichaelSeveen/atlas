ALTER TABLE atlas_identity.memberships
    ADD COLUMN email_hint text;

ALTER TABLE atlas_identity.memberships
    ADD CONSTRAINT memberships_email_hint_masked_length CHECK (
        email_hint IS NULL OR length(email_hint) BETWEEN 3 AND 254
    );

COMMENT ON COLUMN atlas_identity.memberships.email_hint IS
    'Optional already-masked invitation recipient hint. Never stores a canonical or raw email address.';
