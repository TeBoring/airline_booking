package tranlayer

import (
	"P3-f12/official/lsplog"
	"P3-f12/official/transaction"
	"P3-f12/contrib/tranlayerproto"
	"P3-f12/contrib/airlineproto"
	"P3-f12/contrib/libconn"
	"net/rpc"
	"time"
	"fmt"
	"P3-f12/contrib/tranrpc"
)

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
}

//Request Types
const (
	INIT_TRANS = iota
	TRANS_RESPONSE
	TRANS_EXPIRED
	TRANS_OLD
)

type TranLayer struct {
	myhostport string
	activeTransMap map[string](*activeTransMapEle)
	deferTransMap map[string](chan interface{})
	reqChan chan *Request
	oldTransactionMap map[string](*OldTransaction)
	oldLog map[string]int
	lib_conn *libconn.Libconn
}

func NewTranLayer(lc *libconn.Libconn, agencyid, myhostport string) (*TranLayer, error) {
	// Initialize new tranlayer
	tl := &TranLayer{}
	tl.myhostport = myhostport
	tl.activeTransMap = make(map[string](*activeTransMapEle))
	tl.deferTransMap = make(map[string](chan interface{}))
	tl.oldTransactionMap = make(map[string](*OldTransaction))
	tl.oldLog = make(map[string]int)
	tl.lib_conn = lc
	
	trpc := tranrpc.NewTranRPC(tl)
	rpc.Register(trpc)
			
	tl.reqChan = make(chan *Request)
	go tl.requestHandler()
	
	return tl, nil
}

func (tl *TranLayer) BookingFlights(orderList []tranlayerproto.Order) error {
	
	// find server for each airline company
	tranOrderList := make([]*tranOrder, len(orderList))
	
	for i, order := range orderList {
		conn, pptID, err := tl.lib_conn.GetServerWithAddress(order.AirlineID)
		if lsplog.CheckReport(2, err) {
			return err
		}
		tranOrderList[i] = &tranOrder{order.FlightID, pptID, order.Amount, conn}
	}
	
	
	// get unique transaction id
	tranID := fmt.Sprintf("%s:%d", tl.myhostport, time.Now().UnixNano())
	lsplog.Vlogf(5, "Begin transaction:"+tranID)
	// send request to store handler
	req := &reqContent{INIT_TRANS, tranID, tranOrderList}
	replyc := make(chan interface{})
	tl.reqChan <- &Request{req, replyc}
	// wait for response
	status := (<- replyc).(bool)
	lsplog.Vlogf(5, "End of transaction:" +tranID)
	if status {
		return nil
	}
	return lsplog.MakeErr("Transaction Failed")
}

// RPC
func (tl *TranLayer) TransResponse(args *airlineproto.TranArgs, reply *airlineproto.TranReply) error {
	lsplog.Vlogf(5, args.TranID + ": Reeceived trans respose %d", args.Vote)
	// send request to store handler
	var req *reqContent
	switch args.TranType {
	case transaction.TRANS_RESPONSE:
		req = &reqContent{TRANS_RESPONSE, "", args}
	case transaction.TRANS_OLD:
		req = &reqContent{TRANS_OLD, "", args}
	}
	
	switch req.reqType {
	case TRANS_RESPONSE:
		tl.reqChan <- &Request{req, nil}
		reply.Status = airlineproto.OK
	case TRANS_OLD:
		replyc := make(chan interface{})
		tl.reqChan <- &Request{req, replyc}
		decision := (<-replyc).(int)
		reply.Status = decision
	}
	
	// reply to rpc
	reply.Status = airlineproto.OK
	return nil
}

func (tl *TranLayer) requestHandler() {
	for {
		// send remain transaction orders
		if len(tl.oldTransactionMap) != 0 {
			for transID, oldTransaction := range tl.oldTransactionMap {
				order := oldTransaction.orderList.Remove().(*tranOrder)
				
				flightID := order.flightID
				amount := order.amount
				conn := order.conn
				pptList := oldTransaction.pptList
				
				args := &airlineproto.TranArgs{transaction.TRANS_INIT, transID, flightID, amount, pptList, 0, tl.myhostport}
				var reply airlineproto.TranReply
				conn.Call("AirlineRPC.Transaction", args, &reply)
				
				if oldTransaction.orderList.Empty() {
					delete(tl.oldTransactionMap, transID)
					go tl.transTimer(transID)
				}
				
				break
			}
			continue
		}
	
		request := <- tl.reqChan
		lsplog.Vlogf(6, "Handling request")
		switch request.content.reqType {
		case INIT_TRANS, TRANS_RESPONSE, TRANS_EXPIRED, TRANS_OLD:
			tl.transactionHandler(request)
		}
	}
}
