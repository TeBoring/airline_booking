export GOPATH=$(pwd)/..
PROJECT_PATH=$GOPATH

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
