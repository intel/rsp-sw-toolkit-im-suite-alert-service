#!/bin/bash
./generateSecrets.sh
sudo docker stack deploy --compose-file docker-compose-dev.yml RRP-Alert --with-registry-auth