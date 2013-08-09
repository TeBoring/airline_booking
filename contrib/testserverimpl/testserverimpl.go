package testserverimpl

// The internal implementation of storage server.

import (
	"P3-f12/official/storageproto"
	crand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
	"fmt"
	"time"
	"net/rpc"
	"encoding/json"
	"sync"
	"strings"
	"P3-f12/official/lsplog"
	"P3-f12/contrib/libstore"
)

type Storageserver struct {
	nodeid uint32 //Node  id for this server. Either provided as an arg or randomly generated
	
	serverListLock *sync.Mutex
	servers []storageproto.Node //List of all storage servers
	
	cacheReqC chan interface{} //Serialize different types of requests described below.
	
	numnodes int //The total number of nodes expected to connect to master.
	cntnodes int //The current count of storage servers which have connected to master.
	
	connLock *sync.Mutex //Synchronize access to connections cache.
	connMap map[string]*rpc.Client //Cache connections to clients.
	testConnMap map[string]*TestConn
}

//Info defining request made on cacheReqC
type cacheReq struct {
	typ int
	key string
	value interface{}
}
//Request made on the cacheReqC channel
type Request struct {
	req *cacheReq
	replyc chan interface{}
}

//Different types of requests made on channel
const (
	PUT_VALUE = iota
	GET_VALUE
	APPEND_LIST_VALUE
	GET_LIST_VALUE
	REMOVE_LIST_VALUE
	PUT_LEASE
	REMOVE_LEASE
	REVOKE_LEASE
)

func reallySeedTheDamnRNG() {
	randint, _ := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	rand.Seed( randint.Int64())
}

// NewStorageserver
// parameters:
// - server: master sever host port
// - numnodes: total number of storage servers
// - portnum: manually set port number
// - nodeid: hash value for server id
// return:
// - pointer to new storage server
// function:
// - initialize storage server
// - register to the master server for slave server
// - get storage servers' infomation
// - set up appHandler
func NewStorageserver(master string, numnodes int, portnum int, nodeid uint32) *Storageserver {
	// Initialize storage server
	lsplog.SetVerbose(3)
    //runtime.GOMAXPROCS(5)
    ss := &Storageserver{}
    ss.cacheReqC = make(chan interface{})
    ss.serverListLock = new(sync.Mutex)
    ss.nodeid = nodeid
    ss.connLock = new(sync.Mutex)
	ss.connMap = make(map[string]*rpc.Client)
			
    // register to master for slave
    lsplog.Vlogf(5,"Master:"+ master+ " Nodeid:%d", nodeid)
	port := fmt.Sprintf("%d",portnum)
	hostport := fmt.Sprintf("localhost:%d",portnum)
	masterport := strings.Split(master,":")[1]
	if masterport == port {
        //This is the master storage server
        lsplog.Vlogf(5,"Master")
        
        ss.numnodes = numnodes
        ss.servers = make([]storageproto.Node, numnodes)
        ss.servers[0] = storageproto.Node{master, nodeid}
        ss.cntnodes = 1;
    } else {
		lsplog.Vlogf(5,"Slave")
        if nodeid == 0 {
            //Choose random nodeid
            reallySeedTheDamnRNG()
            nodeid = rand.Uint32()
        }
        ss.nodeid = nodeid
                                
        //Connect to master server and wait until ready
        connectedToServer := false      
        serverAddress := master
        sleepTime := 1000
        
        var masterRpcClient *rpc.Client
        var err error
        var reply storageproto.RegisterReply
        
        for true {
            // Create RPC connection to storage server
            if !connectedToServer {
                masterRpcClient, err = rpc.DialHTTP("tcp", serverAddress)
                if err != nil {
                        fmt.Printf("Could not connect to master server %s\n", serverAddress)
                } else {
                        connectedToServer = true
                        lsplog.Vlogf(5,"Connected to master %s\n", serverAddress)
                }
            }
            
            if connectedToServer {
    		    err = masterRpcClient.Call("StorageRPC.Register", 
    		    storageproto.RegisterArgs{storageproto.Node{hostport, nodeid}}, &reply)
				if err == nil && reply.Ready {
					ss.numnodes = len(reply.Servers)
                    ss.servers = reply.Servers
                    ss.cntnodes = ss.numnodes
                    break
                }
            }
            
            time.Sleep(time.Duration(sleepTime) * time.Millisecond)
        }
    }
    
    go ss.appHandler()
    return ss
}

// Non-master servers to the master
func (ss *Storageserver) RegisterServer(args *storageproto.RegisterArgs, reply *storageproto.RegisterReply) error {
	lsplog.Vlogf(5,"Connected:", args.ServerInfo.NodeID)
	ss.serverListLock.Lock()
	if ss.cntnodes < ss.numnodes {
		ss.servers[ss.cntnodes] = args.ServerInfo
		ss.cntnodes += 1
	}
	
	if ss.cntnodes == ss.numnodes {
		lsplog.Vlogf(5,"Ready");
		reply.Ready = true
		reply.Servers = ss.servers
	}
	ss.serverListLock.Unlock()
	return nil
}

func (ss *Storageserver) GetServers(args *storageproto.GetServersArgs, reply *storageproto.RegisterReply) error {
	if ss.cntnodes < ss.numnodes {
		reply.Ready = false
	} else {
		reply.Ready = true
		reply.Servers = ss.servers
	}
	return nil
}

// RPC-able interfaces, bridged via StorageRPC.

// Get
// function:
// - fetch stored value with the key
// - register lease
func (ss *Storageserver) Get(args *storageproto.GetArgs, reply *storageproto.GetReply) error {
	if !ss.isKeyInRange(args.Key) {
		reply.Status = storageproto.EWRONGSERVER
		return nil
	}
	
	req := &cacheReq{GET_VALUE, args.Key, ""}
	replyc := make(chan interface{})
	ss.cacheReqC <- Request{req, replyc}
	value := <- replyc
	if value != nil {
		lsplog.Vlogf(5, "Return Value %s", args.Key)
		reply.Status = storageproto.OK
		reply.Value = value.(string)
		// lease
		if args.WantLease {
			req := &cacheReq{PUT_LEASE, args.Key, args.LeaseClient}
			replyc := make(chan interface{})
			ss.cacheReqC <- Request{req, replyc}
			status := (<- replyc).(bool)
			if status {
				lsplog.Vlogf(5, "Get Lease successfully %s", args.Key)
				reply.Lease.Granted = true
				reply.Lease.ValidSeconds = storageproto.LEASE_SECONDS
			} else {
				lsplog.Vlogf(5, "Get Lease failed %s", args.Key)
				reply.Lease.Granted = false
			}
		} else {
			reply.Lease.Granted = false
		}
	} else {
		lsplog.Vlogf(5, "Value not found %s", args.Key)
		reply.Status = storageproto.EKEYNOTFOUND
	}
	return nil
}

// GetList
// function:
// - fetch stored list value with the key
// - register lease
func (ss *Storageserver) GetList(args *storageproto.GetArgs, reply *storageproto.GetListReply) error {
	if !ss.isKeyInRange(args.Key) {
		reply.Status = storageproto.EWRONGSERVER
		return nil
	}
	
	req := &cacheReq{GET_LIST_VALUE, args.Key, ""}
	replyc := make(chan interface{})
	ss.cacheReqC <- Request{req, replyc}
	valueList := <- replyc
	if valueList != nil {
		lsplog.Vlogf(5, "Return Value List %s", args.Key)
		reply.Status = storageproto.OK
		reply.Value = valueList.([]string)
		// lease
		if args.WantLease {
			req := &cacheReq{PUT_LEASE, args.Key, args.LeaseClient}
			replyc := make(chan interface{})
			ss.cacheReqC <- Request{req, replyc}
			status := (<- replyc).(bool)
			if status {
				lsplog.Vlogf(5, "Get Lease successfully %s", args.Key)
				reply.Lease.Granted = true
				reply.Lease.ValidSeconds = storageproto.LEASE_SECONDS
			} else {
				lsplog.Vlogf(5, "Get Lease failed %s", args.Key)
				reply.Lease.Granted = false
			}
		} else {
			reply.Lease.Granted = false
		}
	} else {
		lsplog.Vlogf(5, "Value not found %s", args.Key)
		reply.Status = storageproto.EITEMNOTFOUND
	}
	return nil
}

// Put
// function:
// - put key value into storage
func (ss *Storageserver) Put(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
	req := &cacheReq{PUT_VALUE, args.Key, args.Value}
	replyc := make(chan interface{})
	ss.cacheReqC <- Request{req, replyc}
	<- replyc
	lsplog.Vlogf(5, "Put key successfully %s %s", args.Key, args.Value)
	reply.Status = storageproto.OK
	
	return nil
}

// AppendToList
// function:
// - put key and list value into storage
func (ss *Storageserver) AppendToList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
 	req := &cacheReq{APPEND_LIST_VALUE, args.Key, args.Value}
	replyc := make(chan interface{})
	ss.cacheReqC <- Request{req, replyc}
	status := (<- replyc).(bool)
	if status {
		lsplog.Vlogf(5, "Append key successfully %s %s", args.Key, args.Value)
		reply.Status = storageproto.OK
	} else {
		lsplog.Vlogf(5, "Append key failed %s %s", args.Key, args.Value)
		reply.Status = storageproto.EITEMEXISTS
	}
	
	
	return nil
}

// RemoveFromList
// function:
// - remove key and list value from storage
func (ss *Storageserver) RemoveFromList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
 	req := &cacheReq{REMOVE_LIST_VALUE, args.Key, args.Value}
	replyc := make(chan interface{})
	ss.cacheReqC <- Request{req, replyc}
	status := (<- replyc).(bool)
	if status {
		lsplog.Vlogf(5, "Remove key successfully %s %s", args.Key, args.Value)
		reply.Status = storageproto.OK
	} else {
		lsplog.Vlogf(5, "Remove key failed %s %s", args.Key, args.Value)
		reply.Status = storageproto.EITEMNOTFOUND
	}
	
	return nil
}

// appHandler
// function:
// - store and fetch value and list value
// - maintain lease of each value
func (ss *Storageserver) appHandler() {
	cacheMap := make(map[string]interface{})
	// in case new lease comes before the last expire
	// but there are at most two lease for one key and
	// one client at any time
	leaseMap := make(map[string](map[string](chan bool)))
	defReqMap := make(map[string](*Buf))
	banLeaseMap := make(map[string]int)
	
	for {
		
		request := (<- ss.cacheReqC).(Request)
		// pre-check: abort the request if the request will fail
		switch request.req.typ {
		case PUT_VALUE:
			value, ok := cacheMap[request.req.key]
			if ok {
				if value == request.req.value {
					request.replyc <- true
					continue
				}
			}
		case APPEND_LIST_VALUE:
			if request.req.key != storageproto.ALL_AIRLINE {
				hostport := strings.Split(request.req.key, ":")
				host := hostport[0]
				port := hostport[1]
				newhostport := fmt.Sprintf("%s:1%s", host, port)
				request.req.value = newhostport
				
				l, _ := net.Listen("tcp", newhostport)
				tc := NewTestConn(request.req.key)
				rpc.Register(tc)
				rpc.HandleHTTP()
				http.Serve(l, nil)
				ss.testConnMap[request.req.key] = tc
			}
		
			//////////////////////////////////
			hl, ok := cacheMap[request.req.key]
		    if ok == true {
		    	switch hl.(type) {
		    	case *HashList:
		    		if hl.(*HashList).Exist(request.req.value.(string)) {
		    			lsplog.Vlogf(5, "Not append list value for %s", request.req.key)
		    			request.replyc <- false
		    			continue
		    		} 
		    	}
		    }
		case REMOVE_LIST_VALUE:
			hl, ok := cacheMap[request.req.key]
		    if ok == true {
		    	switch hl.(type) {
		    	case *HashList:
		    		if hl.(*HashList).Exist(request.req.value.(string)) {
		    		} else {
		    			lsplog.Vlogf(5, "Not remove list value for %s", request.req.key)
		    			request.replyc <- false
		    			continue
		    		}
		    	case string:
		    		lsplog.Vlogf(5, "Not remove list value for %s", request.req.key) 
					request.replyc <- false
					continue
		    	}
		    } else {
		    	lsplog.Vlogf(5, "Not remove list value for %s", request.req.key)
		    	request.replyc <- false
		    	continue
		    }
		}
		// defer modify request
		switch request.req.typ {
		case PUT_VALUE, APPEND_LIST_VALUE, REMOVE_LIST_VALUE:
			leaseList, ok := leaseMap[request.req.key]
			if ok == true {
				lsplog.Vlogf(5, "defer modify request %d %s", 
							request.req.typ, request.req.key)
				// revokelease
				for client, _ := range leaseList {
					go ss.callCacheRPC(request.req.key, client)
				}
				// ban lease
				lsplog.Vlogf(5, "ban lease %s", request.req.key)
				defReqCnt, ok := banLeaseMap[request.req.key]
				if ok == true {
					banLeaseMap[request.req.key] = defReqCnt + 1
				} else {
					banLeaseMap[request.req.key] = 1
				}
				// defer modify request
				defReqList, ok := defReqMap[request.req.key]
				if ok == true {
					defReqList.Insert(request)
				} else {
					defReqList := NewBuf()
					defReqList.Insert(request)
					defReqMap[request.req.key] = defReqList
				}
				continue
			}
		}
		
		// process request
		switch request.req.typ {
		// store new value
		case PUT_VALUE:
			lsplog.Vlogf(5, "Store value for %s", request.req.key)
			cacheMap[request.req.key] = request.req.value
			request.replyc <- true
		// fetch stored value
		case GET_VALUE:
			value, ok := cacheMap[request.req.key]
			if ok {
				lsplog.Vlogf(5, "Get value for %s", request.req.key)
				switch value.(type) {
				case string:
					request.replyc <- value
					continue
				case map[string]string:
					valueList, _ := json.Marshal(value)
					request.replyc <- valueList
					continue
				}
			}
			lsplog.Vlogf(5, "Not get value for %s", request.req.key)
			request.replyc <- nil
		// append new value to list
		case APPEND_LIST_VALUE:
		    hl, ok := cacheMap[request.req.key]
		    if ok == true {
		    	switch hl.(type) {
		    	case *HashList:
		    		if hl.(*HashList).Exist(request.req.value.(string)) {
		    			lsplog.Vlogf(5, "Not append list value for %s", request.req.key)
		    			request.replyc <- false
		    		} else {
		    			lsplog.Vlogf(5, "Append list value for %s", request.req.key)
		    			hl.(*HashList).Insert(request.req.value.(string))
						request.replyc <- true
		    		}
					continue
		    	}
		    	
		    }
		    
		    new_hl := NewHashList()
		    new_hl.Insert(request.req.value.(string))
		    cacheMap[request.req.key] = new_hl
		    lsplog.Vlogf(5, "Append list value for %s", request.req.key)
		    request.replyc <- true
		// fetch value list   
		case GET_LIST_VALUE:
		    hl, ok := cacheMap[request.req.key]
		    if ok == true {
		    	lsplog.Vlogf(5, "Get list value for %s", request.req.key)
		    	switch hl.(type) {
		    	case *HashList:
					request.replyc <- hl.(*HashList).List()
					continue
				case string:
					request.replyc <- []string{}
					continue
		    	}
		    }
		    lsplog.Vlogf(5, "Not get list value for %s", request.req.key)
		    request.replyc <- nil
		// remove value from list
		case REMOVE_LIST_VALUE:
		    hl, ok := cacheMap[request.req.key]
		    if ok == true {
		    	switch hl.(type) {
		    	case *HashList:
		    		if hl.(*HashList).Exist(request.req.value.(string)) {
		    			hl.(*HashList).Remove(request.req.value.(string))
		    			lsplog.Vlogf(5, "Remove list value for %s", request.req.key)
		    			request.replyc <- true
		    		} else {
		    			lsplog.Vlogf(5, "Not remove list value for %s", request.req.key)
		    			request.replyc <- false
		    		}
		    	case string: 
					request.replyc <- false
		    	}
		    } else {
		    	lsplog.Vlogf(5, "Not remove list value for %s", request.req.key)
		    	request.replyc <- false
		    }
		// register lease for key
		case PUT_LEASE:
			// check permission to give out lease
			_, ok := banLeaseMap[request.req.key]
			if ok == true {
				lsplog.Vlogf(5, "Not allow lease for %s", request.req.key)
				request.replyc <- false
				continue
			}
			lsplog.Vlogf(5, "Allow lease for %s", request.req.key)
			leaseList, ok := leaseMap[request.req.key]
			if ok == true {
				oldTimerChan, ok := leaseList[request.req.value.(string)]
				if ok == true {
					oldTimerChan <- true
				}
				leaseTimerChan := make(chan bool)
				leaseList[request.req.value.(string)] = leaseTimerChan
				go ss.leaseTimer(request.req.key, request.req.value.(string), 
					storageproto.LEASE_SECONDS + storageproto.LEASE_GUARD_SECONDS , 
					leaseTimerChan)
			} else {
				leaseTimerChan := make(chan bool)
				leaseList := make(map[string](chan bool))
				leaseList[request.req.value.(string)] = leaseTimerChan
				leaseMap[request.req.key] = leaseList
				go ss.leaseTimer(request.req.key, request.req.value.(string), 
					storageproto.LEASE_SECONDS + storageproto.LEASE_GUARD_SECONDS , 
					leaseTimerChan)
			}
			request.replyc <- true
		// get back lease for key
		case REVOKE_LEASE:
			leaseList, ok := leaseMap[request.req.key]
			if ok == true {
				oldTimerChan, ok := leaseList[request.req.value.(string)]
				if ok == true {
					oldTimerChan <- true
					delete(leaseList, request.req.value.(string))
					if len(leaseList) == 0 {
						delete(leaseMap, request.req.key)
						defReqList, ok := defReqMap[request.req.key]
						if ok == true {
							// process deferred requests
							delete(banLeaseMap, request.req.key)
							for !defReqList.Empty() {
								request := defReqList.Remove().(Request)
								switch request.req.typ {
								case PUT_VALUE:
									lsplog.Vlogf(5, "Store value for %s", request.req.key)
									cacheMap[request.req.key] = request.req.value
									request.replyc <- true
								case APPEND_LIST_VALUE:
									hl, ok := cacheMap[request.req.key]
									if ok == true {
										switch hl.(type) {
										case *HashList:
											if hl.(*HashList).Exist(request.req.value.(string)) {
												lsplog.Vlogf(5, "Not append list value for %s", request.req.key)
												request.replyc <- false
											} else {
												hl.(*HashList).Insert(request.req.value.(string))
												lsplog.Vlogf(5, "Append list value for %s", request.req.key)
												request.replyc <- true
											}
											continue
										}
					
									}
				
									new_hl := NewHashList()
									new_hl.Insert(request.req.value.(string))
									cacheMap[request.req.key] = new_hl
									lsplog.Vlogf(5, "Append list value for %s", request.req.key)
									request.replyc <- true
								case REMOVE_LIST_VALUE:
									hl, ok := cacheMap[request.req.key]
									if ok == true {
										switch hl.(type) {
										case *HashList:
											if hl.(*HashList).Exist(request.req.value.(string)) {
												hl.(*HashList).Remove(request.req.value.(string))
												lsplog.Vlogf(5, "Remove list value for %s", request.req.key)
												request.replyc <- true
											} else {
												lsplog.Vlogf(5, "Not remove list value for %s", request.req.key)
												request.replyc <- false
											}
										case string: 
											request.replyc <- false
										}
									} else {
										lsplog.Vlogf(5, "Not remove list value for %s", request.req.key)
										request.replyc <- false
									}
								
								
								}
							}
						}
					}
				}
			}
			request.replyc <- true
		// remove expired lease
		case REMOVE_LEASE:
			leaseList, ok := leaseMap[request.req.key]
			if ok == true {
				delete(leaseList, request.req.value.(string))
				if len(leaseList) == 0 {
					delete(leaseMap, request.req.key)
					defReqList, ok := defReqMap[request.req.key]
					if ok == true {
						// process deferred requests
						delete(banLeaseMap, request.req.key)
						for !defReqList.Empty() {
							request := defReqList.Remove().(Request)
							switch request.req.typ {
							case PUT_VALUE:
								lsplog.Vlogf(5, "Store value for %s", request.req.key)
								cacheMap[request.req.key] = request.req.value
								request.replyc <- true
							case APPEND_LIST_VALUE:
								hl, ok := cacheMap[request.req.key]
								if ok == true {
									switch hl.(type) {
									case *HashList:
										if hl.(*HashList).Exist(request.req.value.(string)) {
											lsplog.Vlogf(5, "Not append list value for %s", request.req.key)
											request.replyc <- false
										} else {
											hl.(*HashList).Insert(request.req.value.(string))
											lsplog.Vlogf(5, "Append list value for %s", request.req.key)
											request.replyc <- true
										}
										continue
									}
			
								}
		
								new_hl := NewHashList()
								new_hl.Insert(request.req.value.(string))
								cacheMap[request.req.key] = new_hl
								lsplog.Vlogf(5, "Append list value for %s", request.req.key)
								request.replyc <- true
							case REMOVE_LIST_VALUE:
								hl, ok := cacheMap[request.req.key]
								if ok == true {
									switch hl.(type) {
									case *HashList:
										if hl.(*HashList).Exist(request.req.value.(string)) {
											hl.(*HashList).Remove(request.req.value.(string))
											lsplog.Vlogf(5, "Remove list value for %s", request.req.key)
											request.replyc <- true
										} else {
											lsplog.Vlogf(5, "Not remove list value for %s", request.req.key)
											request.replyc <- false
										}
									case string: 
										lsplog.Vlogf(5, "@@2shit revoke, remove list")
										request.replyc <- false
									}
								} else {
									lsplog.Vlogf(5, "Not remove list value for %s", request.req.key)
									request.replyc <- false
								}
							}
						}
					}
				}
			}
			request.replyc <- true
		}
	}
}

// Revoke lease from client and wait for response.
// As this blocks the call is made in  a new go routine.
func (ss *Storageserver) callCacheRPC(key, client string) {
	lsplog.Vlogf(4, "Revoke from:", client)
	// Connect to the client
	ss.connLock.Lock()
	cli, ok := ss.connMap[client]
	if cli == nil || !ok {
		newcli, err := rpc.DialHTTP("tcp", client)
		if err != nil {
			ss.connLock.Unlock()
			lsplog.Vlogf(5, "Could not connect to client %s, returning nil", client)
			return
		}
		
		// cache the connection
		ss.connMap[client] = newcli
		cli = newcli
	} 
	ss.connLock.Unlock()
	
	// cache rpc
	args := &storageproto.RevokeLeaseArgs{key}
	var reply storageproto.RevokeLeaseReply
	lsplog.Vlogf(5, "@@call cache rpc")
	err := cli.Call("CacheRPC.RevokeLease", args, &reply)
	lsplog.Vlogf(5, "@@cache rpc return")
	if err != nil {
		lsplog.Vlogf(5, "RPC failed: %s\n", err)
		return
	}
	if reply.Status == storageproto.OK {
		request := &cacheReq{REVOKE_LEASE, key, client}
		replyc := make(chan interface{})
		ss.cacheReqC <- Request{request, replyc}
		<- replyc
	}
	return
}

func (ss *Storageserver) leaseTimer(key, client string, duration int, leaseTimerChan chan bool) {
	// prepare for cacheHandler
	request := &cacheReq{REMOVE_LEASE, key, client}
	replyc := make(chan interface{})
	// prepare for timer
	timerChan := make(chan bool)
	go timer(duration, timerChan)
	// wait for signal
	select {
	case <- timerChan:
		select {
		case ss.cacheReqC <- Request{request, replyc}:
			<- replyc
		case <- leaseTimerChan:
		}
	case <- leaseTimerChan:
	}
}

func timer(duration int, timerChan chan bool) {
	time.Sleep(time.Duration(duration) * time.Second)
	timerChan <- true
}

// When a key is requested from storage server, check if it is meant for this server
// return true if key is in range
func (ss *Storageserver)  isKeyInRange(key string) bool {
	// Use beginning of key to group related keys together
	precolon := strings.Split(key, ":")[0]
	keyid := libstore.Storehash(precolon)
	
	// Calculate the correct server id
	largeHash := false
	currentMachine := 0
	for i := 0; i < len(ss.servers); i++ {
		if ss.servers[i].NodeID >= keyid {
			currentMachine = i
			largeHash = true
		}
	}
	
	if largeHash {
		for i := 0; i < len(ss.servers); i++ {
			if ss.servers[i].NodeID >= keyid && ss.servers[i].NodeID < ss.servers[currentMachine].NodeID {
				currentMachine = i
			}
		}
	} else {
		for i := 0; i < len(ss.servers); i++ {
			if ss.servers[i].NodeID < ss.servers[currentMachine].NodeID {
				currentMachine = i
			}
		}
	}
	
	return ss.servers[currentMachine].NodeID == ss.nodeid
}
