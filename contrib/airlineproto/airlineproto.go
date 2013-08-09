package airlineproto

// Status codes
const (
	OK = iota
	EFLIGHTNOTFOUND
	ETRANSFAIL
	EFLIGHTEXISTS // create flight
)

// Transaction types
/*const (
	PREPARE = iota
	COMMIT
	ABORT
)*/

// Leasing
const (
	QUERY_CACHE_SECONDS = 10 // Time period for tracking queries
	QUERY_CACHE_THRESH  = 3  // if 3 queries in QUERY_CACHE_SECONDS, get lease
	LEASE_SECONDS       = 10 // Leases are valid for LEASE_SECONDS seconds
	LEASE_GUARD_SECONDS = 2  // And the server should wait an extra 2
)

type FlightInfo struct {
	ID       string
	Departure string
	From     string
	To       string
	Capacity int
	Slot     int
	Cost     int
}

type LeaseStruct struct {
	Granted bool
	ValidSeconds int
}

type TranArgs struct {
	TranType int // PREPARE/COMMIT/ABORT
	TranID   string
	FlightID string
	Amount   int
	PptList []string
	Vote int
	TM string
}

type TranReply struct {
	//TranID   string
	//FlightID string
	Status int
}

type GetArgs struct {
	FlightID    string
	WantLease   bool
	LeaseClient string // host:port of client that wants lease, for callback
}

type GetReply struct {
	Status     int
	FlightInfo FlightInfo
	Lease LeaseStruct
}

type GetAllArgs struct {
	From string
	To string
	Departure string
}

type GetAllReply struct {
	Status     int
	FlightIDs []string
}

//type GetListReply struct {
//	Status int
//	Value []string
//	Lease LeaseStruct
//}

type PutArgs struct {
	FlightID string
	Info FlightInfo
}

type PutReply struct {
	Status int
}

//type Node struct {
//	HostPort string
//	NodeID uint32
//}
//
//type RegisterArgs struct {
//	ServerInfo Node
//}
//
//// RegisterReply is sent in response to both Register and GetServers
//type RegisterReply struct {
//	Ready bool
//	Servers []Node 
//}
//
//type GetServersArgs struct {
//}
//
//// Used by the Cacher RPC
//type RevokeLeaseArgs struct {
//	Key string
//}
//
//type RevokeLeaseReply struct {
//	Status int
//}
