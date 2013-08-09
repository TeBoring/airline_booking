package airlineimpl

import (
	//"strings"
	"P3-f12/contrib/airlineproto"
	"P3-f12/official/transaction"
	//"P3-f12/official/storageproto"
	"P3-f12/official/lsplog"
	"P3-f12/official/bufi"
	//"time"
	//"net/http"
	//"net/rpc"
	//"net"
	"strings"
	"log"
	"fmt"
	"P3-f12/contrib/libconn"
	"P3-f12/contrib/libpaxos"
)



//Different types of requests made on channel
const (
	CREATE_FLIGHT = iota
	GET_FLIGHT
	GET_ALL_FLIGHTS
	PUT_LEASE
	INIT_TRANS
	TRANS_RESPONSE
	TRANS_EXPIRED
	TRANS_OLD
	TRANS_OLD_RESPONSE
	PAXOS_REQUEST
)

// Different types of transaction log
const (
	START = iota
	UPDATE
	END
)

type transLog struct {
	logType  int
	transID  string
	flightID string
	old      int
	new      int
	pptList  []string
	tm       string
}

//Info defining request made on reqChan
type reqContent struct {
	reqType int
	key     string
	value   interface{}
}

//Request made on the cacheReqC channel
type Request struct {
	content *reqContent
	replyc  chan interface{}
	own bool
}

type AirlineServer struct {
	cmdNO int
	reqChan   chan *Request
	airlineID string
	myhostport string
	//ownStorage *rpc.Client
	logBuf    map[string](*bufi.Buf)
	flightBuf map[string](*airlineproto.FlightInfo)
	flightQueryBuf map[string](string)
	lockBuf   map[string](bool)
	oldLog map[string]int
	lib_conn *libconn.Libconn
	oldReplycMap map[int](chan interface{})
	lp *libpaxos.Libpaxos
}

func NewAirlineServer(airlineID, myhostport, masterStorage, ownStorage string, replicas []string, pos int) (*AirlineServer, *libpaxos.Libpaxos, error) {
	lsplog.SetVerbose(4)
	as := &AirlineServer{}
	as.cmdNO = 0
	as.reqChan = make(chan *Request)
	as.airlineID = airlineID
	as.myhostport = myhostport
	as.logBuf = make(map[string](*bufi.Buf))
	as.flightBuf = make(map[string](*airlineproto.FlightInfo))
	as.flightQueryBuf = make(map[string](string))
	as.lockBuf = make(map[string](bool))
	as.oldLog = make(map[string]int)
	as.oldReplycMap = make(map[int](chan interface{}))
	
	q := len(replicas) / 2 + len(replicas) % 2
	as.lp = libpaxos.NewLibpaxos(myhostport, replicas, q, pos, as)
	
	lc, err := libconn.NewLibconn(airlineID, masterStorage, myhostport, 0)
	if lsplog.CheckReport(2, err) {
		return nil, nil, err
	}
	as.lib_conn = lc
	
//	
//	// Create RPC connection to own storage server
//	conn2, err := rpc.DialHTTP("tcp", ownStorage)
//	if err != nil {
//		return nil, err
//	}
//	as.ownStorage = conn2
//	
	// Declare existense
	lc.PublishAirline()
	lc.DeclareExistence()
	
	go as.storeHandler()
	log.Printf("[%s:%s] Start", airlineID, myhostport)
	return as, as.lp, nil
}

// Paxos Function
func (as *AirlineServer) PaxosCreateFlight(args *airlineproto.PutArgs, oldReplyc chan interface{}, own bool) error {
	// send request to store handler
	req := &reqContent{CREATE_FLIGHT, args.FlightID, &args.Info}
	replyc := make(chan interface{})
	as.reqChan <- &Request{req, replyc, own}
	
	// process answer
	status := (<- replyc).(bool)
	if own == true {
		oldReplyc <- status
	}
	return nil
}

func (as *AirlineServer) PaxosGetFlight(args *airlineproto.GetArgs, oldReplyc chan interface{}, own bool) error {
	// send request to store handler
	req := &reqContent{GET_FLIGHT, args.FlightID, nil}
	replyc := make(chan interface{})
	as.reqChan <- &Request{req, replyc, own}
	
	// process answer
	info := <- replyc
	if own == true {
		oldReplyc <- info
	}
	return nil
}

func (as *AirlineServer) PaxosGetAllFlightIDs(args *airlineproto.GetAllArgs, oldReplyc chan interface{}, own bool) error {
	// send request to store handler
	req := &reqContent{GET_ALL_FLIGHTS, getAllFlightsKey(args.From, args.To, args.Departure), nil}
	replyc := make(chan interface{})
	as.reqChan <- &Request{req, replyc, own}
	// process answer
	list := (<- replyc).([]string)
	if own == true {
		oldReplyc <- list
	}
	return nil
}

// asynchronous
func (as *AirlineServer) PaxosTransaction(args *airlineproto.TranArgs, oldReplyc chan interface{}, own bool) error {
	// send request to store handler
	var req *reqContent
	switch args.TranType {
	case transaction.TRANS_INIT:
		req = &reqContent{INIT_TRANS, "", args}
	case transaction.TRANS_RESPONSE:
		req = &reqContent{TRANS_RESPONSE, "", args}
	case transaction.TRANS_OLD:
		req = &reqContent{TRANS_OLD, "", args}
	case transaction.TRANS_OLD_RESPONSE:
		req = &reqContent{TRANS_OLD_RESPONSE, "", args}
	}
	
	switch req.reqType {
	case INIT_TRANS, TRANS_RESPONSE, TRANS_OLD_RESPONSE:
		as.reqChan <- &Request{req, nil, own}
	case TRANS_OLD:
		replyc := make(chan interface{})
		as.reqChan <- &Request{req, replyc, own}
		decision := (<-replyc).(int)
		if own == true {
			oldReplyc <- decision
		}
	}
	
	return nil
	
}


// BookTicket(key: tranID#flightID, value: num)

func (as *AirlineServer) storeHandler() {
	
	
	for {
	
		request := <-as.reqChan
		
		// process request
		switch request.content.reqType {
		case PAXOS_REQUEST:
			paxosCmd := fmt.Sprintf("%d@%s", as.cmdNO, request.content.key)
			as.oldReplycMap[as.cmdNO] = request.replyc
			as.cmdNO++
			var reply libpaxos.Reply
			as.lp.ProposeCommand(paxosCmd, &reply)
			
		//////////////////////////////////////////
		case INIT_TRANS, TRANS_RESPONSE, TRANS_EXPIRED:
			as.transactionHandler(request)
		case CREATE_FLIGHT:
			flightID := request.content.key
			info := request.content.value.(*airlineproto.FlightInfo)
			
			_, exist := as.getFlightInfo(flightID)
			if exist == true {
				log.Printf("[%s:%s] Fail to create new flight: %s", as.airlineID, as.myhostport, flightID)
				request.replyc <- false
			} else {
				log.Printf("[%s:%s] Create new flight: %s", as.airlineID, as.myhostport, flightID)
				as.flightBuf[flightID] = info
				
				//Build additional map to support flight queries
				queryKey := getAllFlightsKey(info.From, info.To, info.Departure)
				vq, eq := as.flightQueryBuf[queryKey]
				if eq {
					as.flightQueryBuf[queryKey] = vq+FLIGHTID_SEP+flightID
				} else {
					as.flightQueryBuf[queryKey] = flightID
				}
				request.replyc <- true
				
			}
			
			
		case GET_FLIGHT:
			flightID := request.content.key
			
			info, exist := as.getFlightInfo(flightID)
			
			if exist == true {
				log.Printf("[%s:%s]  Get flight info for %s", as.airlineID, as.myhostport, flightID)
				request.replyc <- info
				
			} else {
				log.Printf("[%s:%s]  Fail to get flight info for %s", as.airlineID, as.myhostport, flightID)
				request.replyc <- nil
				
			}
			
		case GET_ALL_FLIGHTS:
			queryKey := request.content.key
			vq, eq := as.flightQueryBuf[queryKey]
			
			if eq {
				log.Printf("[%s:%s]  Get flight infos", as.airlineID, as.myhostport)
				request.replyc <- strings.Split(vq, FLIGHTID_SEP)
			} else {
				log.Printf("[%s:%s] Fail to get flight infos", as.airlineID, as.myhostport)
				request.replyc <- []string {}
			}
		}
	}
}

/*
 * Local Functions
 */
 
 func (as *AirlineServer) getFlightInfo(key string) (*airlineproto.FlightInfo, bool) {
 	info, exist := as.flightBuf[key]
	return info, exist
}
 
 func (as *AirlineServer) getFlightLock(key string) bool {
	_, exist := as.lockBuf[key]
	if exist == true {
		return false
	}
	as.lockBuf[key] = true
 	return true
 }
 

//Utility
const SEP string = ":"
func getAllFlightsKey(from, to, date string) string {
	return from+SEP+to+SEP+date
}

const FLIGHTID_SEP string = ";"
