#!/usr/bin/env bash

# derived from:
# https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh

BINARIES=("maru2" "maru2-publish")
REPO_URL="https://github.com/defenseunicorns/maru2"

: ${USE_SUDO:="true"}
: ${MARU2_INSTALL_DIR:="/usr/local/bin"}

# relies upon .goreleaser.yaml following uname conventions
ARCH=$(uname -m)
OS=$(uname)

# runs the given command as root (detects if we are root already)
runAsRoot() {
  local CMD="$*"

  if [ $EUID -ne 0 -a $USE_SUDO = "true" ]; then
    CMD="sudo $CMD"
  fi

  $CMD
}

# scurl invokes `curl` with secure defaults
scurl() {
  # - `--proto =https` requires that all URLs use HTTPS. Attempts to call http://
  #   URLs will fail.
  # - `--tlsv1.2` ensures that at least TLS v1.2 is used, disabling less secure
  #   prior TLS versions.
  # - `--fail` ensures that the command fails if HTTP response is not 2xx.
  # - `--show-error` causes curl to output error messages when it fails (when
  #   also invoked with -s|--silent).
  if [[ "$DEBUG" == "true" ]]; then
    echo "Executing: curl --proto \"=https\" --tlsv1.2 --fail --show-error $*" >&2
  fi
  curl --proto "=https" --tlsv1.2 --fail --show-error "$@"
}

# verifySupported checks that the os/arch combination is supported for
# binary builds.
verifySupported() {
  local supported="Darwin_arm64\nDarwin_x86_64\nLinux_arm64\nLinux_x86_64"
  if ! echo "${supported}" | grep -q "${OS}_${ARCH}"; then
    echo "No prebuilt binary for ${OS}_${ARCH}."
    echo "To build from source, go to $REPO_URL"
    exit 1
  fi

  if ! type "curl" > /dev/null && ! type "wget" > /dev/null; then
    echo "Either curl or wget is required"
    exit 1
  fi
}

# checkMaru2InstalledVersion checks which version of maru2 is installed and
# if it needs to be changed.
checkMaru2InstalledVersion() {
  if [[ -f "${MARU2_INSTALL_DIR}/${BINARIES[0]}" ]]; then
    local version
    version=$(maru2 --version)
    if [[ "$version" == "$TAG" ]]; then
      echo "maru2 ${version} is already ${DESIRED_VERSION:-latest}"
      return 0
    else
      echo "maru2 ${TAG} is available. Changing from version ${version}."
      return 1
    fi
  else
    return 1
  fi
}

# checkTagProvided checks whether TAG has provided as an environment variable so we can skip checkLatestVersion.
checkTagProvided() {
  [[ ! -z "$TAG" ]]
}

# checkLatestVersion grabs the latest version string from the releases
checkLatestVersion() {
  local latest_release_url="$REPO_URL/releases/latest"
  if type "curl" > /dev/null; then
    TAG=$(scurl -Ls -o /dev/null -w "%{url_effective}" $latest_release_url | grep -oE "[^/]+$" )
  elif type "wget" > /dev/null; then
    TAG=$(wget $latest_release_url --server-response -O /dev/null 2>&1 | awk '/^\s*Location: /{DEST=$2} END{ print DEST}' | grep -oE "[^/]+$")
  fi
  if [[ "$DEBUG" == "true" ]]; then
    echo "Resolved latest tag: <$TAG>" >&2
  fi
  if [[ "$TAG" == "latest" ]]; then
    echo "Failed to get the latest version for $REPO_URL"
    exit 1
  fi
}

# downloadTarball downloads the requested release tarball
downloadTarball() {
  MARU2_DIST="maru2_${OS}_$ARCH.tar.gz"
  DOWNLOAD_URL="$REPO_URL/releases/download/$TAG/$MARU2_DIST"
  MARU2_TMP_ROOT="$(mktemp -dt maru2-binary-XXXXXX)"
  MARU2_TMP_FILE="$MARU2_TMP_ROOT/$MARU2_DIST"
  echo "Fetching $DOWNLOAD_URL"
  if type "curl" > /dev/null; then
    scurl -sL "$DOWNLOAD_URL" -o "$MARU2_TMP_FILE"
  elif type "wget" > /dev/null; then
    wget -q -O "$MARU2_TMP_FILE" "$DOWNLOAD_URL"
  fi
}

# installBinaries extracts and installs the binaries
installBinaries() {
  if [[ ! -d "$MARU2_INSTALL_DIR" ]]; then
    echo "Error: Install directory '$MARU2_INSTALL_DIR' does not exist. Please create it first."
    exit 1
  fi

  MARU2_EXTRACT_DIR="$MARU2_TMP_ROOT/extract"
  mkdir -p "$MARU2_EXTRACT_DIR"
  tar -xzf "$MARU2_TMP_FILE" -C "$MARU2_EXTRACT_DIR"

  for binary in "${BINARIES[@]}"; do
    runAsRoot cp "$MARU2_EXTRACT_DIR/$binary" "$MARU2_INSTALL_DIR/"
    runAsRoot chmod +x "$MARU2_INSTALL_DIR/$binary"
    echo "$binary $TAG installed into $MARU2_INSTALL_DIR"
  done
}

# fail_trap is executed if an error occurs.
fail_trap() {
  result=$?
  if [ "$result" != "0" ]; then
    if [[ -n "$INPUT_ARGUMENTS" ]]; then
      echo "Failed to install with the arguments provided: $INPUT_ARGUMENTS"
      help
    else
      echo "Failed to install"
    fi
    echo -e "\tFor support, go to $REPO_URL"
  fi
  cleanup
  exit $result
}

# testVersion tests the installed client to make sure it is working.
testVersion() {
  for binary in "${BINARIES[@]}"; do
    if ! command -v "$binary" &> /dev/null; then
      echo "$binary not found. Is $MARU2_INSTALL_DIR on your "'$PATH?'
      exit 1
    fi
  done
}

# help provides possible cli installation arguments
help () {
  echo "Accepted cli arguments are:"
  echo -e "\t[--help|-h ] ->> prints this help"
  echo -e "\t[--no-sudo]  ->> install without sudo"
}

# cleanup temporary files
cleanup() {
  if [[ -d "${MARU2_TMP_ROOT:-}" ]]; then
    rm -rf "$MARU2_TMP_ROOT"
  fi
}

# Execution

#Stop execution on any error
trap "fail_trap" EXIT
set -e

# Parsing input arguments (if any)
export INPUT_ARGUMENTS="${*}"
set -u
while [[ $# -gt 0 ]]; do
  case $1 in
    '--no-sudo')
       USE_SUDO="false"
       ;;
    '--help'|-h)
       help
       exit 0
       ;;
    *) exit 1
       ;;
  esac
  shift
done
set +u

verifySupported
checkTagProvided || checkLatestVersion
if ! checkMaru2InstalledVersion; then
  downloadTarball
  installBinaries
fi
testVersion
cleanup