package airlinerpc

import (
	"P3-f12/contrib/airlineproto"
)

type AirlineInterface interface {
	/*RegisterServer(*storageproto.RegisterArgs, *storageproto.RegisterReply) error
	GetServers(*storageproto.GetServersArgs, *storageproto.RegisterReply) error
	Get(*storageproto.GetArgs, *storageproto.GetReply) error
	GetList(*storageproto.GetArgs, *storageproto.GetListReply) error
	Put(*storageproto.PutArgs, *storageproto.PutReply) error
	AppendToList(*storageproto.PutArgs, *storageproto.PutReply) error
	RemoveFromList(*storageproto.PutArgs, *storageproto.PutReply) error
	*/
	CreateFlight(*airlineproto.PutArgs, *airlineproto.PutReply) error
	GetFlight(*airlineproto.GetArgs, *airlineproto.GetReply) error
	GetAllFlightIDs(*airlineproto.GetAllArgs, *airlineproto.GetAllReply) error
	Transaction(*airlineproto.TranArgs, *airlineproto.TranReply) error
}

type AirlineRPC struct {
	as AirlineInterface
}

func NewAirlineRPC(as AirlineInterface) *AirlineRPC {
	return &AirlineRPC{as}
}

func (arpc *AirlineRPC) CreateFlight(args *airlineproto.PutArgs, reply *airlineproto.PutReply) error {
	return arpc.as.CreateFlight(args, reply)
}

func (arpc *AirlineRPC) GetFlight(args *airlineproto.GetArgs, reply *airlineproto.GetReply) error {
	return arpc.as.GetFlight(args, reply)
}

func (arpc *AirlineRPC) GetAllFlightIDs(args *airlineproto.GetAllArgs, reply *airlineproto.GetAllReply) error {
	return arpc.as.GetAllFlightIDs(args, reply)
}

func (arpc *AirlineRPC) Transaction(args *airlineproto.TranArgs, reply *airlineproto.TranReply) error {
	return arpc.as.Transaction(args, reply)
}

/*func (srpc *StorageRPC) Get(args *storageproto.GetArgs, reply *storageproto.GetReply) error {
	return srpc.ss.Get(args, reply)
}

func (srpc *StorageRPC) GetList(args *storageproto.GetArgs, reply *storageproto.GetListReply) error {
	return srpc.ss.GetList(args, reply)
}

func (srpc *StorageRPC) Put(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
	return srpc.ss.Put(args, reply)
}

func (srpc *StorageRPC) AppendToList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
	return srpc.ss.AppendToList(args, reply)
}

func (srpc *StorageRPC) RemoveFromList(args *storageproto.PutArgs, reply *storageproto.PutReply) error {
	return srpc.ss.RemoveFromList(args, reply)
}

func (srpc *StorageRPC) Register(args *storageproto.RegisterArgs, reply *storageproto.RegisterReply) error {
	return srpc.ss.RegisterServer(args, reply)
}

func (srpc *StorageRPC) GetServers(args *storageproto.GetServersArgs, reply *storageproto.RegisterReply) error {
	return srpc.ss.GetServers(args, reply)
}*/
