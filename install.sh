#!/usr/bin/env bash

set -o errexit  \
    -o nounset  \
    -o pipefail

function check_for_go () {
    if [ -n "${GOROOT:-}" ] && which go >/dev/null 2>/dev/null; then
        echo "Found go compiler." >&2
    elif [[ -f "/etc/profile.d/modules.sh" ]]; then
        echo "No go compiler found, trying a module setup..." >&2
        source /etc/profile.d/modules.sh
        module purge
        module load compilers/go
    else
        echo "Could not get a go compiler, exiting..." >&2
        exit 1
    fi
}

echo "Checking go environment..." >&2
check_for_go

echo "Changing into \"$(dirname -- "$0")\"..." >&2
cd "$(dirname -- "$0")"

if [[ ! -r go.sum ]]; then
    if [[ ! -r go.mod ]]; then
        echo "Could not find go.mod file, regenerating..." >&2
        module_path="$(git config --get remote.origin.url)"
        module_path="${module_path#git@}"
        module_path="${module_path#https://}"
        module_path="${module_path/://}"
        module_path="${module_path%.git}"
        go mod init "$module_path"
        go mod tidy
    fi
    echo "Could not find go.sum file, regenerating..." >&2
    go mod download all
fi

echo "Building..." >&2
./build.sh

install_path="${INSTALL_PATH:-/shared/ucl/apps/cluster-bin}"
echo "Installing to: $install_path" >&2
cp -vf bin/* "$install_path"/

echo "Done." >&2
