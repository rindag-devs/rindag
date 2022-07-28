#!/usr/bin/env bash

docker run -d \
  -it \
  --rm \
  --privileged \
  --shm-size=1g \
  -p 5051:5051 \
  rindag-judge-server \
  -enable-grpc \
  -file-timeout 30m
