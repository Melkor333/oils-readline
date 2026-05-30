#!/bin/bash
set -e

# Run in a container
apt update && apt install -y wget gcc g++

VERSION=0.37.0

# Step 1: Set up a working directory
WORKDIR=$(mktemp -d) # Use mktemp to create a temporary directory

# Step 2: Download the tarball
TARBALL_URL="https://oils.pub/download/oils-for-unix-$VERSION.tar.gz" # Update with the correct URL if necessary
TARBALL_PATH="$WORKDIR/oils-for-unix.tar.gz"
wget -O "$TARBALL_PATH" "$TARBALL_URL" # Use curl or wget to fetch the tarball

TARGET=$(pwd)
mkdir -p "$TARGET/assets/"
# Step 3: Extract the tarball
tar -xzf "$TARBALL_PATH" -C "$WORKDIR" # Extract the files into the temporary directory
cd "$WORKDIR/oils-for-unix-$VERSION" # Change into the extracted directory

# Step 4: Configure the build
./configure --without-readline --prefix="../../assets"

# Step 5: Build and install
./build/static-oils.sh
cp _bin/cxx-opt-sh/oils-for-unix-static.stripped "$TARGET/assets/"
# Sometimes this doesn't happen
chmod +x "$TARGET/assets/oils-for-unix-static.stripped"
cd "$TARGET"
# Step 6: Cleanup
# Remove the temporary directory
rm -rf "$WORKDIR"
