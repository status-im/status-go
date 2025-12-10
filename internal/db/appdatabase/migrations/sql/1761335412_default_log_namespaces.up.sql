UPDATE log_config
SET
    log_namespaces = 'wakunode:info'
WHERE
    log_namespaces IS NULL
    OR log_namespaces = '';
