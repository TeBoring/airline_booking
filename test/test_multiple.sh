#!/bin/bash

# Assumes GOPATH only has one path for your project
export GOPATH=$(pwd)/..
PROJECT_PATH=$GOPATH

# Port
STORAGE_PORT=9000
AIRLINE_PORT=8001
AGENCY_PORT=7001

# Build storage server
cd ${PROJECT_PATH}/src/P3-f12/official/storageserver
go build storageserver.go
cd - > /dev/null

# Build airline server
cd ${PROJECT_PATH}/src/P3-f12/contrib/airlineserver
go build airlineserver.go
cd - > /dev/null

# Build tribble server
cd ${PROJECT_PATH}/src/P3-f12/official/tribserver
go build tribserver.go
cd - > /dev/null

# Build client
cd ${PROJECT_PATH}/src/P3-f12/official/tribclient
go build tribclient.go
cd - > /dev/null

cd ${PROJECT_PATH}/src/P3-f12/official/runclient
go build client.go
cd - > /dev/null

# Start master storage server
${PROJECT_PATH}/src/P3-f12/official/storageserver/storageserver -port=${STORAGE_PORT} &
MASTER_STORAGE_SERVER_PID=$!
sleep 5
echo Master Storage Estalbished
MASTER_STORAGE_PORT=$STORAGE_PORT
((STORAGE_PORT++))

# Start airline server 1
${PROJECT_PATH}/src/P3-f12/official/storageserver/storageserver -port=${STORAGE_PORT} &
AIRLINE_STORAGE_SERVER_PID=$!
sleep 5
AIRLINE_STORAGE_PORT=$STORAGE_PORT
((STORAGE_PORT++))

${PROJECT_PATH}/src/P3-f12/contrib/airlineserver/airlineserver -port=${AIRLINE_PORT} -masterstore=${MASTER_STORAGE_PORT} -ownstore=${AIRLINE_STORAGE_PORT} -airline=JetBlue &
AIRLINE_SERVER_PID=$!
sleep 5
echo JetBlue Airline Storage Estalbished
((AIRLINE_PORT++))

# Start airline server 2
${PROJECT_PATH}/src/P3-f12/official/storageserver/storageserver -port=${STORAGE_PORT} &
AIRLINE_STORAGE2_SERVER_PID=$!
sleep 5
AIRLINE_STORAGE2_PORT=$STORAGE_PORT
((STORAGE_PORT++))

${PROJECT_PATH}/src/P3-f12/contrib/airlineserver/airlineserver -port=${AIRLINE_PORT} -masterstore=${MASTER_STORAGE_PORT} -ownstore=${AIRLINE_STORAGE2_PORT} -airline=KLM &
AIRLINE_SERVER2_PID=$!
sleep 5
echo KLM Airline Storage Estalbished
((AIRLINE_PORT++))

#Start trib server
${PROJECT_PATH}/src/P3-f12/official/storageserver/storageserver -port=${STORAGE_PORT} &
AGENCY_STORAGE_SERVER_PID=$!
sleep 5
AGENCY_STORAGE_PORT=$STORAGE_PORT
((STORAGE_PORT++))
echo agency1 Agency Storage Established

${PROJECT_PATH}/src/P3-f12/official/tribserver/tribserver -agency=agency1 -masterstore=${MASTER_STORAGE_PORT} -ownstore=${AGENCY_STORAGE_PORT} -port=${AGENCY_PORT} &
TRIB_SERVER_PID=$!
sleep 5
echo agency1 Agency Server Established
AGENCY_PORT_1=$AGENCY_PORT
((AGENCY_PORT++))

echo --All servers are ready--

# Run client commands
# Add flights
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fa JetBlue 223 d2 f t 1 50
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fa KLM 224 d f t2 2 50  
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fa JetBlue 225 d f2 t 1 50

${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fa JetBlue 123 d f t 1 50
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fa KLM 124 d f t 2 50  
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fa JetBlue 125 d f t 1 50

${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} uc user1
#${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} bl user1
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fv f t d
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} ba user1 JetBlue-123
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} bl user1
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} ba user1 KLM-124 JetBlue-125
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fv f t d
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} bl user1
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} uc user2
echo This transaction should fail
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} ba user2 KLM-124 JetBlue-125
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} bl user2
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fv f t d
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} ba user2 KLM-224 JetBlue-223 KLM-124
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} bl user2
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fv f t d


kill -9 ${TRIB_SERVER_PID}
kill -9 ${AGENCY_STORAGE_SERVER_PID}
kill -9 ${AIRLINE_SERVER2_PID}
kill -9 ${AIRLINE_STORAGE2_SERVER_PID}
kill -9 ${AIRLINE_SERVER_PID}
kill -9 ${AIRLINE_STORAGE_SERVER_PID}
kill -9 ${MASTER_STORAGE_SERVER_PID}
