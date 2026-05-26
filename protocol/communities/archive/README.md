History archive build toggles:
  USE_TORRENT=true enables build tag: use_torrent
  USE_LOGOS_STORAGE=true enables build tag: use_logos_storage
  USE_LOGOS_STORAGE=true also wires libstorage include/lib/runtime paths

Backend-specific test targets:
  make test-torrent
  make test-storage

Make variables:
  USE_LOGOS_STORAGE             (default: false)
  USE_TORRENT                   (default: false)
  LOGOS_STORAGE_SOURCE_DIR      (source-build default: ../logos-storage-nim)
  LOGOS_STORAGE_LIB_DIR         (default: $(LOGOS_STORAGE_SOURCE_DIR)/build unless provided, e.g. by Nix)
  LOGOS_STORAGE_INC_DIR         (default: $(LOGOS_STORAGE_SOURCE_DIR)/library unless provided, e.g. by Nix)
  FUNCTIONAL_TESTS_BUILD_TAGS   (default: gowaku_no_rln)

Examples:
  make test-unit USE_LOGOS_STORAGE=true
  make test-unit USE_TORRENT=true
  make test-unit USE_LOGOS_STORAGE=true USE_TORRENT=true
  make fetch-storage
  make build-storage
  make test-torrent
  make test-storage
  USE_LOGOS_STORAGE=true ./scripts/run_functional_tests.sh
  USE_TORRENT=true ./scripts/run_functional_tests.sh
