// Contrib implementation of libstore
package libstore

import (
	"P3-f12/official/lsplog"
	"P3-f12/official/storageproto"
	"P3-f12/official/cacherpc"
	"net/rpc"
	"strings"
	"time"
	"sync"
	//"log"
)

type Libstore struct {
	/** List of nodes fetched from the server
	* Used for routing messages using consistent hashing
	*/
	nodelist []storageproto.Node
	flags int //Flags determining when to request leases
	myhostport string //Hostport where tribserver was established
	
	/* Cache for frequently accessed keys. */
	cacheReqC chan *CacheReq // Serialize different request types desctibed below
	cacheLock *sync.RWMutex // Synchronize access to cacheMap
	cacheMap map[string]*CacheEntry
	
	/* Map to cache connections */
	connLock *sync.Mutex
	connMap map[int]*rpc.Client
	
	/* Channel to listen for revoke requests */
	revokeReqC chan bool
}

//CacheEntry: A single entry in the cache.
type CacheEntry struct {
	value interface{} //contents of the entry - either a string or a list of strings
	revoked bool // revoked or expired
	expiryTime int64
}

type CacheReq struct {
	typ int 
	time int //Time for which entry is valid in ?
	key string
	value interface{}
	replyc chan interface{}
}

//Request Types (CacheReq) made on channel cacheReqC
const (
	CREATE_CACHE = iota
	FETCH_CACHE
	FETCH_CACHE_LIST
	NEW_QUERY
	GARBAGE_COLLECT
	QUERY_COLLECT
)

// iNewLibstore
// parameters:
// - server: master sever host port
// - myhostport: local port to handle cache rpc
// - flags: ALWAYS->always ask lease
// return:
// - pointer to new libstore
// function:
// - initialize libstore
// - create connection to the master server
// - get storage servers' infomation
// - set up cachHandler, garbage collector timer and cache expiration timer
func iNewLibstore(agencyid, server, myhostport string, flags int) (*Libstore, error) {
	// Initialize new libstore
	ls := &Libstore{}
	ls.flags = flags
	ls.myhostport = myhostport
	ls.cacheReqC = make(chan *CacheReq)
	
	// Create RPC connection to storage server
	cli, err := rpc.DialHTTP("tcp", server)
	if err != nil {
		return nil, err
	}

	// Get list of storage servers from master storage server
	args := storageproto.GetServersArgs{}
	var reply storageproto.RegisterReply
	
	// Try to connect to storage server for at most 5 times
	try := 5
	sleepTime := 2000
	for i := 0; ; i++ {
		err = cli.Call("StorageRPC.GetServers", args, &reply)
		
		if err == nil && reply.Ready {
			ls.cacheLock = new(sync.RWMutex)
			ls.cacheMap = make(map[string]*CacheEntry)
			ls.connLock = new(sync.Mutex)
			ls.connMap = make(map[int]*rpc.Client)
			ls.nodelist = reply.Servers
			
			crpc := cacherpc.NewCacheRPC(ls)
			rpc.Register(crpc)
			
			for i := 1; i < len(ls.nodelist); i++ {
				if server == ls.nodelist[i].HostPort {
					ls.connLock.Lock()
					ls.connMap[i] = cli
					ls.connLock.Unlock()
					break
				}
			}
	
			// succeeded
			go ls.cacheHandler()
			go ls.cacheTimer(2*storageproto.LEASE_SECONDS, GARBAGE_COLLECT)
			go ls.cacheTimer(storageproto.QUERY_CACHE_SECONDS, QUERY_COLLECT)
			
			return ls, nil
		}
		// failed
		if i == try - 1 {
			if err != nil {
				return nil, err
			}
			return nil, lsplog.MakeErr("Storage system not ready")
		}
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
	}
	
	// Impossible to come here
	return ls, nil
}

// getServer
// parameters:
// - key: the key for entry
// return
// - point of the connection to server
// - error
// function:
// - calculate the correct server id to connect
// - connect to the server and cache the connection
func (ls *Libstore) getServer(key string) (*rpc.Client, error) {
	// Use beginning of key to group related keys together
	precolon := strings.Split(key, ":")[0]
	keyid := Storehash(precolon)
	//_ = keyid // use keyid to compile
	
	// Calculate the correct server id
	largeHash := false
	currentMachine := 0
	for i := 0; i < len(ls.nodelist); i++ {
		if ls.nodelist[i].NodeID >= keyid {
			currentMachine = i
			largeHash = true
		}
	}
	
	if largeHash {
		for i := 0; i < len(ls.nodelist); i++ {
			if ls.nodelist[i].NodeID >= keyid && ls.nodelist[i].NodeID < ls.nodelist[currentMachine].NodeID {
				currentMachine = i
			}
		}
	} else {
		for i := 0; i < len(ls.nodelist); i++ {
			if ls.nodelist[i].NodeID < ls.nodelist[currentMachine].NodeID {
				currentMachine = i
			}
		}
	}
	
	// Connect to the storage server
	ls.connLock.Lock()
	cli, ok := ls.connMap[currentMachine]
	if cli == nil || !ok {
		newcli, err := rpc.DialHTTP("tcp", ls.nodelist[currentMachine].HostPort)
		if err != nil {
			ls.connLock.Unlock()
			return nil, err
		}
		// cache the connection
		lsplog.Vlogf(7, "Get new connection to %s", ls.nodelist[currentMachine].HostPort)
		lsplog.Vlogf(7, "Cached new connection to %s", ls.nodelist[currentMachine].HostPort)
		ls.connMap[currentMachine] = newcli
		cli = newcli
	} else {
		lsplog.Vlogf(7, "Get connection to %s from connection cache", ls.nodelist[currentMachine].HostPort)
	}
	ls.connLock.Unlock()
	return cli, nil
}

// iGet
// parameters:
// - key: the key for entry
// return:
// - value related to the key
// - error
// function:
// - get cached value or ask storage server
// - cache new value
func (ls *Libstore) iGet(key, clusterid string) (string, error) {
	// Check cache first
	wantlease := false
	if ls.myhostport != "" {
		// register query
		if ls.flags == ALWAYS_LEASE {
			wantlease = true
		} else {
			replyc := make(chan interface{})
			req := &CacheReq{NEW_QUERY, 0, key, "", replyc}
			ls.cacheReqC <- req
			wantlease = (<- replyc).(bool)
		}
	
		// check cache
		replyc := make(chan interface{})
		req := &CacheReq{FETCH_CACHE, 0, key, "", replyc}
		ls.cacheReqC <- req
		cacheValue := (<- replyc)
		if cacheValue != nil {
			return cacheValue.(string), nil
		}
	}
	
	// Query storage server
	args := &storageproto.GetArgs{key, wantlease, ls.myhostport}
	var reply storageproto.GetReply
	cli, err := ls.getServer(key)
	if err != nil {
		return "", err
	}
	err = cli.Call("StorageRPC.Get", args, &reply)
	if err != nil {
		return "", err
	}
	if reply.Status != storageproto.OK {
		return "", lsplog.MakeErr("Get failed:  Storage error")
	}
	
	// Cache new value
	if wantlease {
		if reply.Lease.Granted {
			time := reply.Lease.ValidSeconds
			replyc := make(chan interface{})
			req := &CacheReq{CREATE_CACHE, time, key, reply.Value, replyc}
			ls.cacheReqC <- req
			<- replyc
		}
	}
	
	return reply.Value, nil
}

// iPut
// parameters:
// - key: the key for entry
// - value: value to store
// return:
// - error
// function:
// - send key and value to storage server
func (ls *Libstore) iPut(key, value, clusterid string) error {
	// Put value
	args := &storageproto.PutArgs{key, value}
	var reply storageproto.PutReply
	cli, err := ls.getServer(key)
	if err != nil {
		return err
	}
	err = cli.Call("StorageRPC.Put", args, &reply)
	if err != nil {
		return err
	}
	if reply.Status != storageproto.OK {
		return lsplog.MakeErr("Put failed:  Storage error")
	}
	return nil
}

// iGetList
// parameters:
// - key: the key for entry
// return:
// - value list related to the key
// - error
// function:
// - get cached value or ask storage server
// - cache new value
func (ls *Libstore) iGetList(key, clusterid string) ([]string, error) {
	// Check cache first
	wantlease := false
	if ls.myhostport != "" {
		// register query
		if ls.flags == ALWAYS_LEASE {
			wantlease = true
		} else {
			replyc := make(chan interface{})
			req := &CacheReq{NEW_QUERY, 0, key, "", replyc}
			ls.cacheReqC <- req
			wantlease = (<- replyc).(bool)
		}
		
		// check cache
		replyc := make(chan interface{})
		req := &CacheReq{FETCH_CACHE_LIST, 0, key, "", replyc}
		ls.cacheReqC <- req
		cacheValue := <- replyc
		if cacheValue != nil {
			return cacheValue.([]string), nil
		}
	}
	
	// Query storage server
	args := &storageproto.GetArgs{key, wantlease, ls.myhostport}
	var reply storageproto.GetListReply
	
	cli, err := ls.getServer(key)
	
	if err != nil {
		return nil, err
	}
	err = cli.Call("StorageRPC.GetList", args, &reply)
	if err != nil {
		return nil, err
	}
	if reply.Status != storageproto.OK {
		return nil, lsplog.MakeErr("GetList failed:  Storage error")
	}
	
	// Cache new value
	if wantlease {
		if reply.Lease.Granted {
			time := reply.Lease.ValidSeconds
			replyc := make(chan interface{})
			req := &CacheReq{CREATE_CACHE, time, key, reply.Value, replyc}
			ls.cacheReqC <- req
			<- replyc
		}
	}
	
	return reply.Value, nil
}

// iRemoveFromList
// parameters:
// - key: the key for item
// - removeitem: the item to remove
// return:
// - error
// function:
// - send key and removeitem to storage server
func (ls *Libstore) iRemoveFromList(key, removeitem, clusterid string) error {
	// Remove list value
	args := &storageproto.PutArgs{key, removeitem}
	var reply storageproto.PutReply
	cli, err := ls.getServer(key)
	if err != nil {
		return err
	}
	err = cli.Call("StorageRPC.RemoveFromList", args, &reply)
	if err != nil {
		return err
	}
	if reply.Status != storageproto.OK {
		return lsplog.MakeErr("RemoveFromList failed:  Storage error")
	}
	return nil
}

// iAppendToList
// parameters:
// - key: the key for item
// - newitem: the item to append
// return:
// - error
// function:
// - send key and newitem to storage server
func (ls *Libstore) iAppendToList(key, newitem, clusterid string) error {
	// Append list value
	args := &storageproto.PutArgs{key, newitem}
	var reply storageproto.PutReply
	cli, err := ls.getServer(key)
	
	if err != nil {
		return err
	}
	err = cli.Call("StorageRPC.AppendToList", args, &reply)
	if err != nil {
		return err
	}
	if reply.Status != storageproto.OK {
		return lsplog.MakeErr("AppendToList failed:  Storage error")
	}
	return nil
}
// RevokeLease
// parameters:
// - args: refer to storageproto.RevokeLeaseArgs
// - reply: refer to storageproto.RevokeLeaseReply
// return:
// - error
// function:
// - set the revoked tag of corresponding entry
func (ls *Libstore) RevokeLease(args *storageproto.RevokeLeaseArgs, reply *storageproto.RevokeLeaseReply) error {
	// Revoke lease
	ls.cacheLock.RLock()
	cacheEntry := ls.cacheMap[args.Key]
	if cacheEntry != nil {
		cacheEntry.revoked = true
	}
	ls.cacheLock.RUnlock()
	reply.Status = storageproto.OK
 	return nil   
}

// cacheHandler
// function:
// - store and fetch cache for value and list value
// - record recent quries and collect old query records
// - routine garbage colloect
func (ls *Libstore) cacheHandler() {
	queryMap := make(map[string]int)
	
	for {
		// get new request
		request := <- ls.cacheReqC
		
		// acquire write lock
		acquireWriteLock := false
		switch request.typ {
		case CREATE_CACHE, NEW_QUERY, QUERY_COLLECT, GARBAGE_COLLECT:
			acquireWriteLock = true
		}
		if acquireWriteLock {
			ls.cacheLock.Lock()
		}
		
		switch request.typ {
		// create cahce for new value or list value
		case CREATE_CACHE:
			lsplog.Vlogf(7, "Create value cache for %s", request.key)
			expiryTime := time.Now().Add(time.Duration(request.time)*time.Second)
			cacheEntry := &CacheEntry{request.value, false, expiryTime.UnixNano()}
			ls.cacheMap[request.key] = cacheEntry
			request.replyc <- true
		// collect expired or revoked cahce for value or list value
		case GARBAGE_COLLECT:
			lsplog.Vlogf(7, "Garbage collect")
			for key, cacheEntry := range ls.cacheMap {
				if cacheEntry.revoked || cacheEntry.expiryTime < time.Now().UnixNano() {
					delete(ls.cacheMap, key)
				}
			}
			request.replyc <- true
		// collect expired query record
		case QUERY_COLLECT:
			lsplog.Vlogf(7, "Query collect")
			for key, value := range queryMap {
				if value == 1 {
					delete(queryMap, key)
				} else {
					queryMap[key] = value - 1
				}
			}
			request.replyc <- true
		// get cache for value
		case FETCH_CACHE:
			cacheEntry, ok := ls.cacheMap[request.key]
			// mark expired cache
			if ok && !cacheEntry.revoked && cacheEntry.expiryTime < time.Now().UnixNano() {
				cacheEntry.revoked = true
			}
			// return cache if exist and not revoked or expired
			if ok && !cacheEntry.revoked {
				switch cacheEntry.value.(type) {
				case string:
					lsplog.Vlogf(7, "Hit value cache for %s", request.key)
					request.replyc <- cacheEntry.value
				case []string:
					lsplog.Vlogf(7, "Miss value cache for %s", request.key)
					request.replyc <- nil
				} 
			} else {
				lsplog.Vlogf(7, "Miss value cache for %s", request.key)
				request.replyc <- nil
			}
		// get cache for list value
		case FETCH_CACHE_LIST:
			cacheEntry, ok := ls.cacheMap[request.key]
			// mark expired cache
			if ok && !cacheEntry.revoked && cacheEntry.expiryTime < time.Now().UnixNano() {
				cacheEntry.revoked = true
			}
			// return cache if exist and not revoked or expired
			if ok && !cacheEntry.revoked {
				switch cacheEntry.value.(type) {
				case string:
					lsplog.Vlogf(7, "Miss list cache for %s", request.key)
					request.replyc <- nil
				case []string:
					lsplog.Vlogf(7, "Hit list cache for %s", request.key)
					request.replyc <- cacheEntry.value
				} 
				
			} else {
				lsplog.Vlogf(7, "Miss list cache for %s", request.key)
				request.replyc <- nil
			}
		// create cahce for new query and decide whether to ask for lease
		case NEW_QUERY:
			lsplog.Vlogf(7, "Create query cache for %s", request.key)
			value, ok := queryMap[request.key]
			if ok {
				value += 1
			} else {
				value = 1
			}
			queryMap[request.key] = value
			request.replyc <- (value >= storageproto.QUERY_CACHE_THRESH)
		}
		
		if acquireWriteLock {
			ls.cacheLock.Unlock()
		}
	}
}

// cacheTimer
// parameters:
// - duration: time to wait
// - typ: timer type (garbage collect or query collect)
func (ls *Libstore) cacheTimer(duration int, typ int) {
	for {
		time.Sleep(time.Duration(duration) * time.Second)
		replyc := make(chan interface{})
		req := &CacheReq{typ, 0, "", "", replyc}
		ls.cacheReqC <- req
		<- replyc
	}
}
