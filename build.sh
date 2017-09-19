#!/bin/bash
# rfid-alert-service
set -e

printHelp() {
    echo "will run build the contents of the folder."
    echo
    echo "Usage: ./build.sh"
    echo
    echo "Options:"
    echo "  -h, --help      Show this help dialog"
    exit 0
}

# parameters
for var in "$@"; do
    case "${var}" in
        "-h" | "--help"      ) printHelp;;
    esac
done

echo -e "  \e[2mGo \e[0m\e[94mBuild(ing)...\e[0m"
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo

