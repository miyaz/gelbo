#!/bin/bash

docker build -t gelbo .
CONTAINER_ID=`docker create gelbo`
docker cp $CONTAINER_ID:/app/gelbo .
docker rm $CONTAINER_ID
