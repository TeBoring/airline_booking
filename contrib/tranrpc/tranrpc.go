package tranrpc

import (
	"P3-f12/contrib/airlineproto"
)

type TranInterface interface {
	TransResponse(*airlineproto.TranArgs, *airlineproto.TranReply) error
}

type TranRPC struct {
	t TranInterface
}

func NewTranRPC(tc TranInterface) *TranRPC {
	return &TranRPC{tc}
}

func (trpc *TranRPC) TransResponse(args *airlineproto.TranArgs, reply *airlineproto.TranReply) error {
        return trpc.t.TransResponse(args, reply)
}
