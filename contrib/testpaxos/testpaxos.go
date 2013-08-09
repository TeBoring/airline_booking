package main

import (
	"fmt"
	"flag"
	"net"
	"net/http"
	"net/rpc"
	"log"
	"P3-f12/contrib/libpaxos"
	"strings"
	"time"
	"runtime"
	"strconv"
)

var portnum *int = flag.Int("port", 7000, "port # to listen on")
var nodepos *int = flag.Int("nodepos", 1, "position")
var rep *string = flag.String("rep", "localhost:7000", "replica addresses")

var N = 30

func main() {
	runtime.GOMAXPROCS(5)
	flag.Parse()
	
	self := fmt.Sprintf(":%d", *portnum)
	l, e := net.Listen("tcp", self)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	
	hostport := fmt.Sprintf("localhost:%d", *portnum)
	
	log.Printf("Paxos replica starting on port %d\n", *portnum);
	history := make(map[int] string)
	tc := &TestClient{strconv.Itoa(*portnum), history}
	lp := libpaxos.NewLibpaxos(hostport, strings.Split(*rep, "-"), 2, *nodepos, tc)
	
	run_cmds := true
	//run_cmds = (*nodepos == 1)
	
	
	if(run_cmds) {
		go func() {
			time.Sleep(5*time.Second)
			//st := 2 * time.Second
			st := 100 * time.Millisecond
			for i := 1; i <= N ; i += 1{
				time.Sleep(st)
				var reply libpaxos.Reply
				vs := fmt.Sprintf("(value %d %d)", i, *nodepos)
				lp.ProposeCommand(vs, &reply)
			}
			
			//time.Sleep(60*time.Second)
		} ()
	}
	
	//go func() {
	rpc.Register(lp) 
	rpc.HandleHTTP()
	http.Serve(l, nil)
	//} ()
}

type TestClient struct {
	rep string
	history map[int] string
}

func (tc TestClient) PaxosResponse(cmd string) {
	pos := len(tc.history)
	tc.history[pos] = cmd
	
	log.Printf("[TestClient %s] RESPONSE %d:%s", tc.rep, pos, cmd);
	
	if len(tc.history) >= 3*N {
		log.Printf("[TestClient %s] History", tc.rep);
			
		for i := 0; i<len(tc.history); i+=1 {
			v := tc.history[i]
			log.Printf(" %d:%s", i, v)
		}
	}
} 