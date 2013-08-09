package testserverimpl

type TestConn struct {
	var delay int64
	var lost float64
	var original string
}

func NewTestConn(original string) *TestConn {
	return &TestConn{0, 0.0, original}
}

TransResponse(args *airlineproto.TranArgs, reply *airlineproto.TranReply) error
Transaction(args *airlineproto.TranArgs, reply *airlineproto.TranReply) error
GetAllFlightIDs(args *airlineproto.GetAllArgs, reply *airlineproto.GetAllReply) error
CreateFlight(args *airlineproto.PutArgs, reply *airlineproto.PutReply) error
GetFlight(args *airlineproto.GetArgs, reply *airlineproto.GetReply) error




