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

# Start airline server
${PROJECT_PATH}/src/P3-f12/official/storageserver/storageserver -port=${STORAGE_PORT} &
AIRLINE_STORAGE_SERVER_PID=$!
sleep 5
AIRLINE_STORAGE_PORT=$STORAGE_PORT
((STORAGE_PORT++))

${PROJECT_PATH}/src/P3-f12/contrib/airlineserver/airlineserver -port=${AIRLINE_PORT} -masterstore=${MASTER_STORAGE_PORT} -ownstore=${AIRLINE_STORAGE_PORT} -airline=JetBlue -replicas="localhost:$AIRLINE_PORT" -pos=1 &
AIRLINE_SERVER_PID=$!
sleep 5
echo Airline Server Estalbished
((AIRLINE_PORT++))
((PAXOS_PORT++))

# Start trib server
${PROJECT_PATH}/src/P3-f12/official/storageserver/storageserver -port=${STORAGE_PORT} &
AGENCY_STORAGE_SERVER_PID=$!
sleep 5
AGENCY_STORAGE_PORT=$STORAGE_PORT
((STORAGE_PORT++))

${PROJECT_PATH}/src/P3-f12/official/tribserver/tribserver -agency=agency1 -masterstore=${MASTER_STORAGE_PORT} -ownstore=${AGENCY_STORAGE_PORT} -port=${AGENCY_PORT} &
TRIB_SERVER_PID=$!
sleep 5
echo Trib server Established
AGENCY_PORT_1=$AGENCY_PORT
((AGENCY_PORT++))

echo All servers are ready

# Run client commands
# Add flights
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fa JetBlue 123 d f t 4 50
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fa JetBlue 124 d f t 2 50  
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fa JetBlue 125 d f t 1 50

${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} uc user1
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} bl user1
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fv f t d
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} ba user1 JetBlue-123
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} bl user1
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} ba user1 JetBlue-124 JetBlue-125
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fv f t d

${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} bl user1
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} uc user2
echo This transaction should fail
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} ba user2 JetBlue-124 JetBlue-125
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} bl user2
${PROJECT_PATH}/src/P3-f12/official/runclient/client -port=${AGENCY_PORT_1} fv f t d

kill -9 ${TRIB_SERVER_PID}
kill -9 ${AGENCY_STORAGE_SERVER_PID}
kill -9 ${AIRLINE_SERVER_PID}
kill -9 ${AIRLINE_STORAGE_SERVER_PID}
kill -9 ${MASTER_STORAGE_SERVER_PID}
