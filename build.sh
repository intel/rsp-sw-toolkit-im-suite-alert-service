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
    echo "  -db, --dockerb  Does a docker build and creates image with tag rfid-Alert-service"
    exit 0
}

buildDocker=false
# parameters
for var in "$@"; do
    case "${var}" in
        "-h" | "--help"      ) printHelp;;
        "-db" | "--dockerb"     ) buildDocker=true;;
    esac
done

echo -e "  \e[2mGo \e[0m\e[94mBuild(ing)...\e[0m"
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo
if [[ "${buildDocker}" == true ]]; then
    echo -e "\e[94m making docker image..."
    sudo docker build -t rfid-alert-service .
fi