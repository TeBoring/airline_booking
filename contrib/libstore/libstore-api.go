// DO NOT CHANGE THIS FILE

// Definition of API for libstore 2012.
// This file defines all of the publicly accessible values and functions.
// The actual implementation of these functions is provided in the
// libstore-impl.go file, which you must write.

package libstore

import (
	"hash/fnv"
)

// Debugging mode flags
const (
	NONE = 0
	ALWAYS_LEASE = iota  // Request leases for every Get and GetList
)

//Cluster id to fetch from all airlines
const ALL_AIRLINES string = "ALL"

// NewLibstore creates a new instance of the libstore client (*Libstore),
// telling it to contact _server_ as the master storage server.
// Myhostport is the lease revocation callback port.
//  - If "", the client will never request leases;
//  - If set to a non-zero value (e.g., "localhost:port"), the caller
//    must have an HTTP listener and RPC listener reachable at that port.
// Flags is one of the debugging mode flags from above.
func NewLibstore(agencyid, servers, myhostport string, flags int) (*Libstore, error) {
	return iNewLibstore(agencyid, servers, myhostport, flags)
}

func (ls *Libstore) Get(key, clusterid string) (string, error) {
	return ls.iGet(key, clusterid)
}

func (ls *Libstore) Put(key, value, clusterid string) error {
	return ls.iPut(key, value, clusterid)
}

func (ls *Libstore) GetList(key, clusterid string) ([]string, error) {
	return ls.iGetList(key, clusterid)
}

func (ls *Libstore) RemoveFromList(key, removeitem, clusterid string) error {
	return ls.iRemoveFromList(key, removeitem, clusterid)
}

func (ls *Libstore) AppendToList(key, newitem, clusterid string) error {
	return ls.iAppendToList(key, newitem, clusterid)
}

// Partitioning:  Defined here so that all implementations
// use the same mechanism.

func Storehash(key string) uint32 {
	hasher := fnv.New32()
	hasher.Write([]byte(key))
	return hasher.Sum32()
}
