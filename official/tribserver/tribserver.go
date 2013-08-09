package main


// A simple main that starts your tribble server.
// DO NOT MODIFY THIS FILE

import (
	"fmt"
	"flag"
	"net"
	"net/http"
	"net/rpc"
	"log"
	"P3-f12/contrib/tribimpl"
)


var agency *string = flag.String("agency", "agency1", "Agency Name")
var masterStoragePort *int = flag.Int("masterstore", 9000, "The server to query address of other servers")
var ownStoragePort *int = flag.Int("ownstore", 9001, "Local storage server")
var portnum *int = flag.Int("port", 7000, "port # to listen on")
var prefer *int = flag.Int("prefer", 0, "prefer which server")

func main() {
	flag.Parse()
	l, e := net.Listen("tcp", fmt.Sprintf(":%d", *portnum))
	if e != nil {
		log.Fatal("listen error:", e)
	}
	//log.Printf("Server starting on port %d\n", *portnum);
	ts := tribimpl.NewTribserver(*agency, 
		fmt.Sprintf("localhost:%d", *masterStoragePort), 
		fmt.Sprintf("localhost:%d", *ownStoragePort),
		fmt.Sprintf("localhost:%d", *portnum), *prefer)
		
	if ts != nil {
		rpc.Register(ts)
		rpc.HandleHTTP()
		http.Serve(l, nil)
	} else {
		//log.Printf("Server could not be created\n")
	}
}
