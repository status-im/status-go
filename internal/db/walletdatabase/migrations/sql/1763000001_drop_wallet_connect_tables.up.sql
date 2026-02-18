-- Drop legacy wallet_connect tables; session storage migrated to connector_wc_sessions
DROP TABLE IF EXISTS wallet_connect_sessions;
DROP TABLE IF EXISTS wallet_connect_dapps;
