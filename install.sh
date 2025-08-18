#!/usr/bin/env sh
# This script installs recaller from the GitHub releases.
# It detects the operating system and architecture, downloads the appropriate zip file,
# extracts the binary, and moves it to a folder in your PATH: /usr/local/bin
# Usage:
#   curl -sf https://raw.githubusercontent.com/cybrota/recaller/refs/heads/main/install.sh | sh

set -e

# Determine OS (capitalize first letter as used in the release artifact filename)
OS=$(uname | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)
    PLATFORM="Linux"
    ;;
  darwin)
    PLATFORM="Darwin"
    ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

# Determine architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)
    ARCH="x86_64"
    ;;
  i386|i686)
    ARCH="i386"
    ;;
  armv6l|armv7l)
    ARCH="arm"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Determine the release version.
if [ -z "$RECALLER_VERSION" ] || [ "$RECALLER_VERSION" = "latest" ]; then
  RECALLER_VERSION=$(curl -s https://api.github.com/repos/cybrota/recaller/releases/latest | \
    grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
  if [ -z "$RECALLER_VERSION" ]; then
    echo "Could not determine latest version." >&2
    exit 1
  fi
fi

# Construct the filename based on the goreleaser name_template.
FILE="recaller_${PLATFORM}_${ARCH}.zip"

# Construct the download URL for the GitHub release.
URL="https://github.com/cybrota/recaller/releases/download/${RECALLER_VERSION}/${FILE}"

echo "Downloading recaller version ${RECALLER_VERSION} for ${PLATFORM}/${ARCH}..."
curl -L -o /tmp/${FILE} "${URL}"

# Create a temporary directory for extraction.
EXTRACT_DIR=$(mktemp -d)

echo "Extracting ${FILE}..."
unzip -q -o /tmp/${FILE} -d "${EXTRACT_DIR}"

# Ensure the binary is executable.
chmod +x "${EXTRACT_DIR}/recaller"

# Determine installation directory.
INSTALL_DIR="/usr/local/bin"
if [ ! -w "${INSTALL_DIR}" ]; then
  echo "No write permission for ${INSTALL_DIR}. Attempting to use sudo..."
  SUDO="sudo"
else
  SUDO=""
fi

echo "Installing recaller to ${INSTALL_DIR}..."
$SUDO mv "${EXTRACT_DIR}/recaller" "${INSTALL_DIR}/recaller"

# Clean up
rm -rf "${EXTRACT_DIR}" /tmp/${FILE}

echo "recaller has been installed successfully. Make sure ${INSTALL_DIR} is in your PATH."

exit 0