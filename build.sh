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
    echo "  -db, --dockerb  Does a docker build and creates image with tag rfid-alert-service:latest"
    echo "  -l, --lint      Runs gometalinter against the source code"
}

buildDocker=false
lint=false
# parameters
for var in "$@"; do
    case "${var}" in
        "-h" | "--help"     ) printHelp; exit 0;;
        "-db" | "--dockerb" ) buildDocker=true;;
        "-l" | "--lint"     ) lint=true;;
        *)  printHelp
            printf "\nERROR -- Invalid argument: ${var}\n"
            exit 1;;
    esac
done

echo -e "  \e[2mGo \e[0m\e[94mBuild(ing)...\e[0m"
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo

if [[ "${buildDocker}" == true ]]; then
    echo -e "\e[94m making docker image..."
    docker build -t rfid-alert-service .
fi

if [[ "${lint}" == true ]]; then
    echo -e "\e[94m running gometalinter..."
    gometalinter.v2 --vendor --deadline=120s --disable gotype --config=../../RSP-Inventory-Suite/ci-go-build-image/linter.json ./...
fi
