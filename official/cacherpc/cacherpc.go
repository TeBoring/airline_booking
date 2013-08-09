// Do not modify this file.
//
// Calls into your own implementation's functions.

package cacherpc

import (
	"P3-f12/official/storageproto"
)

type CacherInterface interface {
	RevokeLease(*storageproto.RevokeLeaseArgs, *storageproto.RevokeLeaseReply) error
}

type CacheRPC struct {
	c CacherInterface
}

func NewCacheRPC(cc CacherInterface) *CacheRPC {
	return &CacheRPC{cc}
}

func (crpc *CacheRPC) RevokeLease(args *storageproto.RevokeLeaseArgs, reply *storageproto.RevokeLeaseReply) error {
        return crpc.c.RevokeLease(args, reply)
}
