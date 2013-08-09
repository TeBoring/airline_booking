package libconn

import (
	"sync"
	"net/rpc"
	"P3-f12/official/lsplog"
	"P3-f12/official/storageproto"
	"log"
	"time"
)

/** Communicate with master server.
 * Cache connections
 */
type Libconn struct {
	entityid string
	master string
	myhostport string 
	
	/* Map to cache connections */
	connLock *sync.Mutex
	connMap map[int]*rpc.Client
	
	//Connection to master
	conn *rpc.Client
	
	prefer int
}

//TODO: Cache connections	
func NewLibconn(entityid, master, myhostport string, prefer int) (*Libconn, error) {
	lc := &Libconn{}
	
	lc.connLock = new(sync.Mutex)
	lc.connMap = make(map[int]*rpc.Client)
	lc.entityid = entityid
	lc.master = master
	lc.myhostport = myhostport
	lc.prefer = prefer
	
	//Connect to master node
	cli, err := rpc.DialHTTP("tcp", master)
	if lsplog.CheckReport(2, err) {
		return nil, err
	}
	lc.conn = cli
	
	return lc, nil
}

func (lc *Libconn) DeclareExistence() error {
	return lc.appendList(lc.entityid, lc.myhostport)
}

func (lc *Libconn) PublishAirline() error {
	return lc.appendList(storageproto.ALL_AIRLINE, lc.entityid)
}

func (lc *Libconn) appendList(key, value string) error {
	args := &storageproto.PutArgs{key, value}
	var reply storageproto.PutReply
	err := lc.conn.Call("StorageRPC.AppendToList", args, &reply)
	if lsplog.CheckReport(2, err) {
		//log.Printf("[%s:%s] Airline already declared\n", lc.entityid, lc.myhostport)
		return lsplog.MakeErr("Declare existense failed")
	}
	if reply.Status != storageproto.OK {
		//log.Printf("[%s:%s] Airline already declared\n", lc.entityid, lc.myhostport)
		return lsplog.MakeErr("Declare Existense Failed")
	}
	
	return nil
}

func (lc *Libconn) GetServer(entityId string) (*rpc.Client, error) {
	cli, _, err := lc.GetServerWithAddress(entityId)
	return cli, err
}

func (lc *Libconn) GetServerAt(hostport string) (*rpc.Client, error) {
	//TODO:
	cli, err := rpc.DialHTTP("tcp", hostport)
    if err != nil {
		return nil, lsplog.MakeErr("Connect airline failed")
	}
	return cli, nil
}

func (lc *Libconn) GetServerWithAddress(entityId string) (*rpc.Client, string, error) {
	// Get entity info
	args := &storageproto.GetArgs{entityId, false, ""}
	var reply storageproto.GetListReply
	err := lc.conn.Call("StorageRPC.GetList", args, &reply)
	if err != nil {
		log.Printf("[%s:%s] Cannot connect to master storage\n", lc.entityid, lc.myhostport)
		return nil, "", lsplog.MakeErr("Connect master storage failed")
	}
	if reply.Status != storageproto.OK {
		log.Printf("[%s:%s] Cannot find address for: %s\n", lc.entityid, lc.myhostport, entityId)
		return nil, "", lsplog.MakeErr("Get agency info failed")
	}
	
	if len(reply.Value) == 0 {
		return nil, "", lsplog.MakeErr("GetServer: Empty list from master for airline:"+entityId)
	}
	
	// Create RPC connection to airline server
	pos := 0
	if lc.prefer == 0 {
		pos = int(time.Now().UnixNano()) % len(reply.Value)
	} else {
		pos = lc.prefer % len(reply.Value)
	}
	server := reply.Value[pos]
	cli, err := rpc.DialHTTP("tcp", server)
	if err != nil {
		log.Printf("[%s:%s] Cannot connect to: %s\n", lc.entityid, lc.myhostport, entityId)
		return nil, "", err
	}
	
	return cli, server, nil
}

func (lc *Libconn) GetServers() ([]*rpc.Client, error) {
	// Get airline info
	args := &storageproto.GetArgs{storageproto.ALL_AIRLINE, false, ""}
	var reply storageproto.GetListReply
	err := lc.conn.Call("StorageRPC.GetList", args, &reply)
	if err != nil {
		log.Printf("[%s:%s] Cannot connect to master storage\n", lc.entityid, lc.myhostport)
		return nil, lsplog.MakeErr("Connect master storage failed")
	}
	if reply.Status != storageproto.OK {
		log.Printf("[%s:%s] Cannot find airline list\n", lc.entityid, lc.myhostport)
		return nil, lsplog.MakeErr("Get server list failed")
	}
	
	// Create RPC connection to airline servers
	servers := reply.Value
	conns := make([]*rpc.Client, len(servers))
	i := 0
	
	for _, server := range servers {
		conn, err := lc.GetServer(server)
		if err == nil {
			conns[i] = conn
			i++
		}
	}
	return conns, nil
}
