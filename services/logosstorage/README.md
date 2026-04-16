USE_LOGOS_STORAGE=true enables for builds:
	  - build tag: use_logos_storage
	  - CGO flags for libstorage include/library paths
	  - runtime library path wiring for tests
	
test-storage always runs with logos-storage support enabled.

Variables:
  USE_LOGOS_STORAGE          (default: false, also used by functional tests)
  LOGOS_STORAGE_SOURCE_DIR   (default: ../logos-storage-nim)
  LOGOS_STORAGE_VERSION      (default: $(LOGOS_STORAGE_VERSION))
  LOGOS_STORAGE_LIB_DIR      (default: \$$LOGOS_STORAGE_SOURCE_DIR/build)
  LOGOS_STORAGE_INC_DIR      (default: \$$LOGOS_STORAGE_SOURCE_DIR/library)
  FUNCTIONAL_TESTS_BUILD_TAGS        (default: gowaku_no_rln)

Examples:
  make test-storage
  make test-unit USE_LOGOS_STORAGE=true
  make build-storage
  USE_LOGOS_STORAGE=true ./scripts/run_functional_tests.sh
