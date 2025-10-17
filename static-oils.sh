#!/bin/bash
set -e

VERSION=0.34.0

# Step 1: Set up a working directory
WORKDIR=$(mktemp -d) # Use mktemp to create a temporary directory

# Step 2: Download the tarball
TARBALL_URL="https://oils.pub/download/oils-for-unix-$VERSION.tar.gz" # Update with the correct URL if necessary
TARBALL_PATH="$WORKDIR/oils-for-unix.tar.gz"
wget -O "$TARBALL_PATH" "$TARBALL_URL" # Use curl or wget to fetch the tarball

OLDCWD=$(pwd)
mkdir -p "$OLDCWD/assets/"
# Step 3: Extract the tarball
tar -xzf "$TARBALL_PATH" -C "$WORKDIR" # Extract the files into the temporary directory
cd "$WORKDIR/oils-for-unix-$VERSION" # Change into the extracted directory

# Step 4: Configure the build
./configure --without-readline --prefix="../../assets"

# Step 5: Build and install
./build/static-oils.sh
cp _bin/cxx-opt-sh/oils-for-unix-static.stripped "$OLDCWD/assets/"
cd "$OLDCWD"
# Step 6: Cleanup
# Remove the temporary directory
rm -rf "$WORKDIR"
