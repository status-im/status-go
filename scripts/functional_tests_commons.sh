remove_old_logs() {
  # Cleanup any previous logs
  rm -rf "${logs_path}"

  # Create directories
  mkdir -p "${logs_path}"
}

set_pyenv() {
  venv_path="${root_path}/.venv"

  if [[ -d "${venv_path}" ]]; then
      echo -e "${GRN}Using existing virtual environment${RST}"
  else
      echo -e "${GRN}Creating new virtual environment${RST}"
      python3 -m venv "${venv_path}"
  fi

  source "${venv_path}/bin/activate"

  # Upgrade pip and install requirements
  echo -e "${GRN}Installing dependencies${RST}"
  pip install --upgrade pip &>/dev/null
  pip install -r "${root_path}/requirements.txt" &>/dev/null
}

discover_tests() {
  pytest --collect-only -q -m rpc -c "${root_path}/pytest.ini" "$1"
}

start_services() {
  # Run docker
  echo -e "${GRN}Running status-go external dependencies${RST}"
  docker compose -p ${project_name} ${all_compose_files} up -d --build --remove-orphans
}

remove_old_containers() {
  # Remove any remaining containers if any
  echo -e "${GRN}Removing any remaining containers (if any relevant left)${RST}"
  docker ps -a --filter "name=status-go-func-tests-${identifier}" -q | xargs -r docker rm -f

  # Remove networks
  docker network rm "status-go-func-tests-${identifier}_default" 2>/dev/null || true
}

clean_all_containers() {
  # Stop containers
  echo -e "${GRN}Stopping docker containers${RST}"
  docker compose -p ${project_name} ${all_compose_files} stop

  # Cleanup containers
  echo -e "${GRN}Removing docker containers${RST}"
  docker compose -p ${project_name} ${all_compose_files} down

  remove_old_containers 
}

wait_for_waku_suite_scanner() {
  # Wait for wakufleet-scanner to finish before running tests. If it fails,
  # there's no point in starting tests because wakufleetconfig.json will not
  # be available for status-backend.
  echo -e "${GRN}Waiting for wakufleet-scanner to complete${RST}"
  scanner_container_id=$(docker compose -p ${project_name} ${all_compose_files} ps -q wakufleet-scanner || true)

  if [[ -z "${scanner_container_id}" ]]; then
    echo -e "${RED}wakufleet-scanner service container not found. Make sure docker-compose.waku.yml defines it correctly.${RST}"
    exit 1
  fi

  scanner_exit_code=$(docker wait "${scanner_container_id}")

  if [[ "${scanner_exit_code}" -ne 0 ]]; then
    echo -e "${RED}wakufleet-scanner failed with exit code ${scanner_exit_code}. See logs below:${RST}"
    docker logs "${scanner_container_id}" || true
    echo -e "${RED}Aborting functional tests because wakufleetconfig.json was not generated successfully.${RST}"
    exit 1
  fi

  echo -e "${GRN}wakufleet-scanner completed successfully${RST}"
}

wait_for_services() {
  local timeout="$1"
  local project="status-go-func-tests-$(git rev-parse --short HEAD)"

  echo "Waiting up to ${timeout}s for boot-1 and store to be healthy..."

  for ((i=1; i<=timeout; i++)); do
    printf "\r%02d" "$i"
    local healthy=$(
      docker compose -p "$project" ps --format json \
      | jq -s -r '
          [ .[]
            | select(.Service=="boot-1" or .Service=="store")
            | (.State=="running" and .Health=="healthy")
          ]
          | (length==2 and all)
        '
    )

    if [[ $healthy == "true" ]]; then
      echo -e "\r✅ Services are healthy (after ${i}s)"
      return 0
    fi

    sleep 1
  done

  echo "❌ Services did not become healthy within ${timeout}s"
  exit 1
}

list_tests_and_confirm() {
  local selected_test="${1:+-k $1}"
  echo -e "${GRN}Discovering tests to be run...${RST}"
  collected_output=$(discover_tests "$selected_test")
  test_count=$(echo "$collected_output" | grep -c "^\s*<Function test_.*>$")
  if [ -z "$selected_test" ]; then
    if [ "$test_count" -eq "0" ]; then
      echo -e "${RED}No tests found!${RST}"
      exit 1
    fi
    echo -e "${RED}No test pattern provided. This will run all ${test_count} tests!${RST}"
  else
    # Early exit if no tests found
    if [ "$test_count" -eq "0" ]; then
      echo -e "${RED}No tests found matching: $1${RST}"
      exit 1
    fi
    
    echo -e "${GRN}Found ${test_count} tests matching:${RST} $1"

    # Show the tests that will run
    echo -e "${GRN}Tests to execute:${RST}"
    echo "$collected_output" \
    | grep -oP "^\s*<Function \Ktest_[^>]*(?=>$)" \
    | nl -w2 -s') '
  fi
  
  read -p "Continue with execution? (y/n): " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      exit 0
  fi
}

run_tests() {
  # Decide parallelization strategy based on number of tests matched
  if [ "$test_count" -eq 1 ]; then
      echo -e "${GRN}Running single test without parallelization${RST}"
      parallel_opts=""
  elif [ "$test_count" -le 4 ]; then
      echo -e "${GRN}Running with limited parallelization (-n $test_count)${RST}"
      parallel_opts="-n $test_count --dist loadgroup"
  else
      echo -e "${GRN}Running with full parallelization (-n 12)${RST}"
      parallel_opts="-n 12 --dist loadgroup"
  fi

  local selected_test="${1:+-k $1}"

  # Run with dynamic parallelization
  pytest --reruns 2 -m rpc -c "${root_path}/pytest.ini" $parallel_opts \
    --log-cli-level="${FUNCTIONAL_TESTS_LOG_LEVEL}" \
    --docker_project_name="${project_name}" \
    --docker-image=${image_name} \
    --logs-dir="${logs_path}" ${selected_test}
  exit_code=$?
}

# Set up cleanup trap to ensure containers are always cleaned up
cleanup() {
    echo -e "${YLW}Running cleanup...${RST}"
    clean_all_containers
}

trap cleanup EXIT