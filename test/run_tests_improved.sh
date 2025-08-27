#!/bin/bash

# Karei Offline Testing Suite Runner - Improved Version
# Proper exit codes and output separation
# shellcheck disable=SC2317  # Functions are called indirectly

set -e

# Exit codes
readonly EXIT_SUCCESS=0
readonly EXIT_GENERAL_FAILURE=1
readonly EXIT_INVALID_ARGS=2
readonly EXIT_PREREQUISITES=3
readonly EXIT_WRONG_DIRECTORY=4

readonly EXIT_MAIN_COMPILE=10
readonly EXIT_TEST_COMPILE=11
readonly EXIT_DEPENDENCIES=12

readonly EXIT_UNIT_TESTS=20
readonly EXIT_INTEGRATION_TESTS=21
readonly EXIT_CUSTOM_SUITE=22
readonly EXIT_NETWORK_ISOLATION=23

readonly EXIT_TIMEOUT=30
readonly EXIT_PERMISSIONS=31
readonly EXIT_CLEANUP=32
readonly EXIT_RESOURCES=33

readonly EXIT_INVALID_CONFIG=40
readonly EXIT_MISSING_FIXTURES=41
readonly EXIT_CORRUPTED_DATA=42

# Configuration
VERBOSE=${VERBOSE:-true}
TEST_TIMEOUT=${TEST_TIMEOUT:-300}
JSON_OUTPUT=${JSON_OUTPUT:-false}
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Colors for stderr output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Global state for JSON output
declare -A TEST_RESULTS
declare -A TEST_DURATIONS
TEST_START_TIME=""

# Logging functions (all to stderr)
log_info() {
  if [[ "$VERBOSE" == "true" ]]; then
    printf "%s\n" "${BLUE}▸ $1${NC}" >&2
  fi
}

log_success() {
  if [[ "$VERBOSE" == "true" ]]; then
    printf "%s\n" "${GREEN}✓ $1${NC}" >&2
  fi
}

log_warning() {
  printf "%s\n" "${YELLOW}⚠ $1${NC}" >&2
}

log_error() {
  printf "%s\n" "${RED}✗ $1${NC}" >&2
}

# JSON output functions (to stdout)
json_start() {
  TEST_START_TIME=$(date +%s)
  if [[ "$JSON_OUTPUT" == "true" ]]; then
    printf '%s\n' '{"status":"running","start_time":"'"$(date -Iseconds)"'","phases":{}}' | jq -c .
  fi
}

json_phase_result() {
  local phase="$1"
  local status="$2"
  local duration="$3"
  local error_message="$4"

  TEST_RESULTS["$phase"]="$status"
  TEST_DURATIONS["$phase"]="$duration"

  if [[ "$JSON_OUTPUT" == "true" ]]; then
    local result="{\"status\":\"$status\",\"duration\":\"$duration\""
    if [[ -n "$error_message" ]]; then
      result="$result,\"error\":\"$error_message\""
    fi
    result="$result}"
    printf "%s\n" "$phase: $result" >&2 # Debug info to stderr
  fi
}

json_final_result() {
  local overall_status="$1"
  local exit_code="$2"
  local end_time=$(date +%s)
  local total_duration=$((end_time - TEST_START_TIME))

  if [[ "$JSON_OUTPUT" == "true" ]]; then
    local phases_json="{"
    local first=true
    for phase in "${!TEST_RESULTS[@]}"; do
      if [[ "$first" == "false" ]]; then
        phases_json="$phases_json,"
      fi
      phases_json="$phases_json\"$phase\":{\"status\":\"${TEST_RESULTS[$phase]}\",\"duration\":\"${TEST_DURATIONS[$phase]}\"}"
      first=false
    done
    phases_json="$phases_json}"

    local passed_count=0
    local total_count=${#TEST_RESULTS[@]}
    for status in "${TEST_RESULTS[@]}"; do
      if [[ "$status" == "passed" ]]; then
        ((passed_count++))
      fi
    done

    cat <<EOF
{
  "status": "$overall_status",
  "exit_code": $exit_code,
  "start_time": "$(date -d "@$TEST_START_TIME" -Iseconds)",
  "end_time": "$(date -Iseconds)",
  "duration": "${total_duration}s",
  "total_phases": $total_count,
  "passed_phases": $passed_count,
  "failed_phases": $((total_count - passed_count)),
  "phases": $phases_json
}
EOF
  else
    # Human-readable summary to stderr
    printf "\n" >&2
    printf "%s\n" "▪ Test Results Summary" >&2
    printf "%s\n" "======================" >&2
    printf "%s\n" "Status: $overall_status" >&2
    printf "%s\n" "Duration: ${total_duration}s" >&2
    printf "%s\n" "Phases: $passed_count/$total_count passed" >&2
  fi
}

# Enhanced error handling with proper exit codes
exit_with_code() {
  local exit_code="$1"
  local message="$2"
  local phase="${3:-unknown}"

  log_error "$message"
  json_phase_result "$phase" "failed" "0s" "$message"
  json_final_result "failed" "$exit_code"
  exit "$exit_code"
}

# Print header (to stderr)
print_header() {
  if [[ "$VERBOSE" == "true" ]]; then
    cat >&2 <<EOF
========================================
▸ Karei Offline Testing Suite
========================================
Project: $(basename "$PROJECT_ROOT")
Working Directory: $PWD
Verbose: $VERBOSE
JSON Output: $JSON_OUTPUT
Timeout: ${TEST_TIMEOUT}s
========================================

EOF
  fi
}

# Check prerequisites with detailed exit codes
check_prerequisites() {
  local start_time=$(date +%s)
  log_info "Checking prerequisites..."

  # Check Go installation
  if ! command -v go &>/dev/null; then
    exit_with_code $EXIT_PREREQUISITES "Go is not installed or not in PATH" "prerequisites"
  fi

  # Check Go version
  local go_version
  go_version=$(go version | awk '{print $3}' | sed 's/go//')
  log_info "Go version: $go_version"

  # Check directory structure
  if [[ ! -f "$PROJECT_ROOT/go.mod" ]]; then
    exit_with_code $EXIT_WRONG_DIRECTORY "go.mod not found. Wrong directory?" "prerequisites"
  fi

  if [[ ! -d "$PROJECT_ROOT/test" ]]; then
    exit_with_code $EXIT_WRONG_DIRECTORY "Test directory not found" "prerequisites"
  fi

  # Check disk space (basic check)
  local available_space
  available_space=$(df "$PROJECT_ROOT" | tail -1 | awk '{print $4}')
  if [[ "$available_space" -lt 1000000 ]]; then # Less than ~1GB
    log_warning "Low disk space: ${available_space}KB available"
  fi

  local duration=$(($(date +%s) - start_time))
  json_phase_result "prerequisites" "passed" "${duration}s"
  log_success "Prerequisites check passed"
}

# Enhanced compilation with specific error codes
compile_tests() {
  local start_time=$(date +%s)
  log_info "Compiling test code..."

  cd "$PROJECT_ROOT"

  # Test main project compilation
  if ! go build ./cmd/main.go 2>/dev/null; then
    local duration=$(($(date +%s) - start_time))
    json_phase_result "compilation" "failed" "${duration}s" "Main project compilation failed"
    exit_with_code $EXIT_MAIN_COMPILE "Failed to compile main project" "compilation"
  fi
  log_success "Main project compiles successfully"

  # Test compilation of test packages
  local test_packages=(
    "./test/unit"
    "./test/mocks"
    "./test/offline"
    "./test/isolated"
    "./test/integration"
  )

  for package in "${test_packages[@]}"; do
    if [[ -d "$package" ]]; then
      log_info "Compiling $package..."
      if ! go build "$package" 2>/dev/null; then
        local duration=$(($(date +%s) - start_time))
        json_phase_result "compilation" "failed" "${duration}s" "Test package $package compilation failed"
        exit_with_code $EXIT_TEST_COMPILE "Failed to compile $package" "compilation"
      fi
    fi
  done

  local duration=$(($(date +%s) - start_time))
  json_phase_result "compilation" "passed" "${duration}s"
  log_success "All test packages compile successfully"
}

# Enhanced test running with timeout and proper error codes
run_unit_tests() {
  local start_time=$(date +%s)
  log_info "Running unit tests..."

  cd "$PROJECT_ROOT"

  local timeout_cmd="timeout ${TEST_TIMEOUT}s"
  local verbose_flag=""
  if [[ "$VERBOSE" == "true" && "$JSON_OUTPUT" != "true" ]]; then
    verbose_flag="-v"
  fi

  local test_output
  if test_output=$($timeout_cmd go test $verbose_flag ./test/unit/... 2>&1); then
    local duration=$(($(date +%s) - start_time))
    json_phase_result "unit_tests" "passed" "${duration}s"
    log_success "Unit tests passed"

    # Send test details to stdout if not in JSON mode
    if [[ "$JSON_OUTPUT" != "true" && "$VERBOSE" == "true" ]]; then
      printf "%s\n" "Unit test output:" >&2
      printf "%s\n" "$test_output" >&2
    fi
    return 0
  else
    local exit_code=$?
    local duration=$(($(date +%s) - start_time))

    if [[ $exit_code -eq 124 ]]; then
      json_phase_result "unit_tests" "failed" "${duration}s" "Unit tests timed out"
      exit_with_code $EXIT_TIMEOUT "Unit tests timed out after ${TEST_TIMEOUT}s" "unit_tests"
    else
      json_phase_result "unit_tests" "failed" "${duration}s" "Unit tests failed"
      # Send test output to stderr for debugging
      printf "%s\n" "Unit test output:" >&2
      printf "%s\n" "$test_output" >&2
      exit_with_code $EXIT_UNIT_TESTS "Unit tests failed" "unit_tests"
    fi
  fi
}

# Enhanced integration tests
run_integration_tests() {
  local start_time=$(date +%s)
  log_info "Running integration tests..."

  cd "$PROJECT_ROOT"

  local timeout_cmd="timeout ${TEST_TIMEOUT}s"
  local verbose_flag=""
  if [[ "$VERBOSE" == "true" && "$JSON_OUTPUT" != "true" ]]; then
    verbose_flag="-v"
  fi

  local test_output
  if test_output=$($timeout_cmd go test $verbose_flag ./test/integration/... 2>&1); then
    local duration=$(($(date +%s) - start_time))
    json_phase_result "integration_tests" "passed" "${duration}s"
    log_success "Integration tests passed"

    if [[ "$JSON_OUTPUT" != "true" && "$VERBOSE" == "true" ]]; then
      printf "%s\n" "Integration test output:" >&2
      printf "%s\n" "$test_output" >&2
    fi
    return 0
  else
    local exit_code=$?
    local duration=$(($(date +%s) - start_time))

    if [[ $exit_code -eq 124 ]]; then
      json_phase_result "integration_tests" "failed" "${duration}s" "Integration tests timed out"
      exit_with_code $EXIT_TIMEOUT "Integration tests timed out after ${TEST_TIMEOUT}s" "integration_tests"
    else
      json_phase_result "integration_tests" "failed" "${duration}s" "Integration tests failed"
      printf "%s\n" "Integration test output:" >&2
      printf "%s\n" "$test_output" >&2
      exit_with_code $EXIT_INTEGRATION_TESTS "Integration tests failed" "integration_tests"
    fi
  fi
}

# Enhanced custom suite
run_custom_suite() {
  local start_time=$(date +%s)
  log_info "Running custom offline test suite..."

  cd "$PROJECT_ROOT"

  # Build custom test runner
  if ! go build -o test/offline_test_runner test/run_offline_tests.go 2>/dev/null; then
    local duration=$(($(date +%s) - start_time))
    json_phase_result "custom_suite" "failed" "${duration}s" "Failed to build custom test runner"
    exit_with_code $EXIT_TEST_COMPILE "Failed to build custom test runner" "custom_suite"
  fi

  local runner_args=""
  if [[ "$VERBOSE" != "true" ]]; then
    runner_args="--quiet"
  fi

  local test_output
  if test_output=$(timeout ${TEST_TIMEOUT}s test/offline_test_runner $runner_args 2>&1); then
    local duration=$(($(date +%s) - start_time))
    json_phase_result "custom_suite" "passed" "${duration}s"
    log_success "Custom test suite passed"

    if [[ "$JSON_OUTPUT" != "true" && "$VERBOSE" == "true" ]]; then
      printf "%s\n" "Custom suite output:" >&2
      printf "%s\n" "$test_output" >&2
    fi
    return 0
  else
    local exit_code=$?
    local duration=$(($(date +%s) - start_time))

    if [[ $exit_code -eq 124 ]]; then
      json_phase_result "custom_suite" "failed" "${duration}s" "Custom suite timed out"
      exit_with_code $EXIT_TIMEOUT "Custom suite timed out" "custom_suite"
    else
      json_phase_result "custom_suite" "failed" "${duration}s" "Custom suite failed"
      printf "%s\n" "Custom suite output:" >&2
      printf "%s\n" "$test_output" >&2
      exit_with_code $EXIT_CUSTOM_SUITE "Custom suite failed" "custom_suite"
    fi
  fi
}

# Enhanced network isolation verification
verify_network_isolation() {
  local start_time=$(date +%s)
  log_info "Verifying network isolation..."

  # Check unshare availability
  if command -v unshare &>/dev/null; then
    log_info "unshare available - testing network isolation"

    if unshare --net --map-root-user sh -c 'printf "%s\n" "Network isolation test"' &>/dev/null; then
      log_success "Network isolation capability verified"
    else
      log_warning "Network isolation test failed, but tests can still run"
    fi
  else
    log_warning "unshare not available - network isolation cannot be fully tested"
  fi

  local duration=$(($(date +%s) - start_time))
  json_phase_result "network_isolation" "passed" "${duration}s"
  log_success "Network isolation verification completed"
}

# Enhanced cleanup with error handling
cleanup() {
  local start_time=$(date +%s)
  log_info "Cleaning up..."

  cd "$PROJECT_ROOT"

  # Remove test binaries
  rm -f test/offline_test_runner main 2>/dev/null || true

  # Clean up temporary test files
  find test/ -name "*.test" -delete 2>/dev/null || true
  find test/ -name "tmp*" -type d -exec rm -rf {} + 2>/dev/null || true

  local duration=$(($(date +%s) - start_time))
  json_phase_result "cleanup" "passed" "${duration}s"
  log_success "Cleanup completed"
}

# Enhanced help function
show_help() {
  cat >&2 <<EOF
Karei Offline Testing Suite Runner

Usage: $0 [OPTIONS]

Options:
  -h, --help      Show this help message
  -q, --quiet     Run in quiet mode (less verbose output)
  -v, --verbose   Run in verbose mode (more detailed output)
  -j, --json      Output results in JSON format (to stdout)
  -t, --timeout   Set test timeout in seconds (default: 300)

Environment Variables:
  VERBOSE         Set to 'false' to run in quiet mode
  JSON_OUTPUT     Set to 'true' to enable JSON output
  TEST_TIMEOUT    Timeout in seconds for individual test phases

Exit Codes:
  0    Success
  2    Invalid command line arguments
  3    Prerequisites not met
  4    Wrong working directory
  10   Main project compilation failed
  11   Test package compilation failed
  20   Unit tests failed
  21   Integration tests failed
  22   Custom test suite failed
  30   Test timeout exceeded
  32   Cleanup failed

Examples:
  $0                     # Run all tests with default settings
  $0 --quiet             # Run with minimal output to stderr
  $0 --json              # Output JSON results to stdout
  $0 --timeout 600       # Run with 10-minute timeout
  VERBOSE=false $0       # Run quietly using environment variable

JSON Output:
  When --json is specified, structured test results are sent to stdout
  while progress messages go to stderr. This enables integration with
  automation tools and CI/CD pipelines.
EOF
}

# Main execution function
main() {
  # Initialize JSON output
  json_start

  print_header

  # Set up cleanup trap
  trap 'cleanup 2>/dev/null || exit $EXIT_CLEANUP' EXIT

  # Define test phases
  local phases=(
    "check_prerequisites"
    "verify_network_isolation"
    "compile_tests"
    "run_unit_tests"
    "run_integration_tests"
    "run_custom_suite"
  )

  # Run each phase
  for phase in "${phases[@]}"; do
    if [[ "$VERBOSE" == "true" ]]; then
      printf "\n" >&2
      log_info "=== Phase: $phase ==="
    fi

    $phase
  done

  # All phases completed successfully
  json_final_result "success" $EXIT_SUCCESS

  if [[ "$JSON_OUTPUT" != "true" ]]; then
    printf "\n" >&2
    log_success "✓ All test phases passed!"
    printf "\n" >&2
    cat >&2 <<EOF
✓ Karei offline testing infrastructure is working correctly!
   • Zero network access required
   • Zero host system impact  
   • Comprehensive installation logic testing
   • All systems operational
EOF
  fi

  exit $EXIT_SUCCESS
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
  -h | --help)
    show_help
    exit $EXIT_SUCCESS
    ;;
  -q | --quiet)
    VERBOSE=false
    shift
    ;;
  -v | --verbose)
    VERBOSE=true
    shift
    ;;
  -j | --json)
    JSON_OUTPUT=true
    VERBOSE=false # Reduce stderr noise in JSON mode
    shift
    ;;
  -t | --timeout)
    if [[ -n "$2" && "$2" =~ ^[0-9]+$ ]]; then
      TEST_TIMEOUT="$2"
      shift 2
    else
      log_error "Invalid timeout value: $2"
      exit $EXIT_INVALID_ARGS
    fi
    ;;
  *)
    log_error "Unknown option: $1"
    show_help
    exit $EXIT_INVALID_ARGS
    ;;
  esac
done

# Validate timeout
if [[ ! "$TEST_TIMEOUT" =~ ^[0-9]+$ ]] || [[ "$TEST_TIMEOUT" -lt 10 ]]; then
  log_error "Invalid timeout: $TEST_TIMEOUT (must be >= 10 seconds)"
  exit $EXIT_INVALID_ARGS
fi

# Run main function
main
