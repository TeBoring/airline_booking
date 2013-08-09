#!/bin/bash

# Assumes GOPATH only has one path for your project
export GOPATH=$(pwd)/..
PROJECT_PATH=$GOPATH

# Port
STORAGE_PORT=9000
AIRLINE_PORT=8001
AGENCY_PORT=7001

# Build testpaxos server
cd ${PROJECT_PATH}/src/P3-f12/contrib/testpaxos
go build testpaxos.go
cd - > /dev/null

# Start replicas
${PROJECT_PATH}/src/P3-f12/contrib/testpaxos/testpaxos -port=7000 -nodepos=1 -rep=localhost:7000-localhost:8000-localhost:9000 &
REPLICA_1_PID=$!
echo Replica 1 Estalbished

${PROJECT_PATH}/src/P3-f12/contrib/testpaxos/testpaxos -port=8000 -nodepos=2 -rep=localhost:7000-localhost:8000-localhost:9000 &
REPLICA_2_PID=$!
echo Replica 2 Estalbished

sleep 30

${PROJECT_PATH}/src/P3-f12/contrib/testpaxos/testpaxos -port=9000 -nodepos=3 -rep=localhost:7000-localhost:8000-localhost:9000 &
REPLICA_3_PID=$!
echo Replica 3 Estalbished

sleep 30

kill -9 ${REPLICA_1_PID}
kill -9 ${REPLICA_2_PID}
kill -9 ${REPLICA_3_PID}