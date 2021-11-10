CREATE INDEX encryption_bundles_expired_identity_installation_id_version on bundles (expired, identity, installation_id, version);
CREATE INDEX encryption_bundles_expired_identity_installation_id on bundles (identity, installation_id, version);
CREATE INDEX ratchet_info_v2_identity_installation_id on ratchet_info_v2 (identity, installation_id);
