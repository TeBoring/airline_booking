package libairline

import (
	"P3-f12/contrib/airlineproto"
	"P3-f12/contrib/tranlayerproto"
	"P3-f12/contrib/tranlayer"
	"P3-f12/official/tribproto"
	"P3-f12/official/lsplog"
	"P3-f12/official/bufi"
	"P3-f12/contrib/libconn"
	"log"
	"fmt"
)

/* Communicate with all airline storage servers */
type Libairline struct {
	lib_tranlayer *tranlayer.TranLayer
	lib_conn *libconn.Libconn
	agencyid string
	myhostport string
}

func NewLibairline(lc *libconn.Libconn, agencyid, myhostport string) (*Libairline, error) {
	la := &Libairline{}
	
	la.lib_conn = lc
	la.agencyid = agencyid
	la.myhostport = myhostport
	
	trnlayer, err := tranlayer.NewTranLayer(lc, agencyid, myhostport) 
	if lsplog.CheckReport(2, err) {
		return nil, err
	}
	
	// Declare existense
	err = lc.DeclareExistence()
	if lsplog.CheckReport(2, err) {
		return nil, err
	}
	
	la.lib_tranlayer = trnlayer
	return la, nil
}

func (la *Libairline) GetAllFlights(from, to, dep string) ([]string, error) {
	log.Printf("[%s:%s] Start to get flights from %s to %s, on %s\n", la.agencyid, la.myhostport, from, to, dep)
	conns, err := la.lib_conn.GetServers()
	if err != nil {
		log.Printf("[%s:%s] Fail to get flights from %s to %s, on %s\n", la.agencyid, la.myhostport, from, to, dep)
		return nil, err
	}
	
	
	flightMap := make(map[string]bool)
	for _, conn := range conns {
		args := &airlineproto.GetAllArgs{from, to, dep}
		var reply airlineproto.GetAllReply
		err := conn.Call("AirlineRPC.GetAllFlightIDs", args, &reply)
		if lsplog.CheckReport(2, err) {
			log.Printf("[%s:%s] Fail to get flights from %s to %s, on %s\n", la.agencyid, la.myhostport, from, to, dep)
			return nil, err
		}

		if reply.Status != airlineproto.OK {
			log.Printf("[%s:%s] Fail to get flights from %s to %s, on %s\n", la.agencyid, la.myhostport, from, to, dep)
			return nil, lsplog.MakeErr("Fetch all flights failed")
		}
		for _, flight := range reply.FlightIDs {
			flightMap[flight] = true
		}
	}
	
	flights := make([]string, len(flightMap))
	i := 0
	for flight, _ := range flightMap {
		flights[i] = flight
		i++
	}
	
	log.Printf("[%s:%s] Succeed to get flights from %s to %s, on %s\n", la.agencyid, la.myhostport, from, to, dep)
	return flights, nil
}

func (la *Libairline) GetFlight(flightid string) (airlineproto.FlightInfo, error) {
	conn, err := la.lib_conn.GetServer(tribproto.GetAirlineFromFlightId(flightid))
	if lsplog.CheckReport(2, err) {
		return airlineproto.FlightInfo{}, err
	}
	
	args := &airlineproto.GetArgs{flightid, false, ""}
	var reply airlineproto.GetReply
	err = conn.Call("AirlineRPC.GetFlight", args, &reply)
	if lsplog.CheckReport(2, err) {
		return airlineproto.FlightInfo{}, err
	}
	
	if reply.Status != airlineproto.OK {
		return airlineproto.FlightInfo{}, lsplog.MakeErr("Fetch flight failed")
	}
	
	return reply.FlightInfo, nil
}

func (la *Libairline) CreateFlight(flight airlineproto.FlightInfo) error {
	log.Printf("[%s:%s] Start to add flight: %s\n", la.agencyid, la.myhostport, flight.ID)

	conn, err := la.lib_conn.GetServer(tribproto.GetAirlineFromFlightId(flight.ID))
	if err != nil {
		log.Printf("[%s:%s] Fail to add flight: %s\n", la.agencyid, la.myhostport, flight.ID)
		return lsplog.MakeErr("Airline doesn't exist")
	}
	
	args := &airlineproto.PutArgs{flight.ID, flight}
	var reply airlineproto.PutReply
	err = conn.Call("AirlineRPC.CreateFlight", args, &reply)
	if lsplog.CheckReport(2, err) {
		log.Printf("[%s:%s] Fail to add flight: %s\n", la.agencyid, la.myhostport, flight.ID)
		return lsplog.MakeErr("Flight already exist")
	}
	
	if reply.Status != airlineproto.OK {
		log.Printf("[%s:%s] Fail to add flight: %s\n", la.agencyid, la.myhostport, flight.ID)
		return lsplog.MakeErr("Flight already exist")
	}
	
	log.Printf("[%s:%s] Succeed to add flight: %s\n", la.agencyid, la.myhostport, flight.ID)
	return nil
}

func (la *Libairline) MakeBooking(flightids []string, isRemove bool) error {
	//TODO: Handle remove
	orderQuantity := 1

	if isRemove {
		orderQuantity = -1
	}


	orderGrpMap := make(map[string]*bufi.Buf)
	
	for _, v := range flightids {
		airlineId := tribproto.GetAirlineFromFlightId(v)
		orderList, exist := orderGrpMap[airlineId]
		if exist == false {
			orderList = bufi.NewBuf()
		}
		orderList.Insert(v)
		orderGrpMap[airlineId] = orderList
	}
	
	
	orders := make([]tranlayerproto.Order, len(orderGrpMap))
	i := 0
	for airlineId, orderList := range orderGrpMap {
		flightIds := ""
		for !orderList.Empty() {
			flightId := orderList.Remove().(string)
			if flightIds == "" {
				flightIds = flightId
			} else {
				flightIds = fmt.Sprintf("%s:%s", flightIds, flightId)
			}
		}
	
		order := tranlayerproto.Order{airlineId, flightIds, orderQuantity}
		orders[i] = order
		i++
	}
	err := la.lib_tranlayer.BookingFlights(orders)
	if lsplog.CheckReport(2, err) {
		return err
	}
	
	return nil
}


