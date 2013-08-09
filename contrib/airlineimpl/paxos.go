package airlineimpl

import (
	"P3-f12/contrib/airlineproto"
	"P3-f12/official/transaction"
	"encoding/json"
	"fmt"
	"strings"
	"strconv"
)
//
// Airline Server
//
func (as *AirlineServer) CreateFlight(args *airlineproto.PutArgs, reply *airlineproto.PutReply) error {
	// send request to store handler
	cmdArgs, _ := json.Marshal(*args)
	paxosCmd := fmt.Sprintf("%s@CF@%s", as.myhostport, cmdArgs)
	
	req := &reqContent{PAXOS_REQUEST, paxosCmd, nil}
	replyc := make(chan interface{})
	as.reqChan <- &Request{req, replyc, true}

	// process answer
	status := (<- replyc).(bool)
	if status {
		reply.Status = airlineproto.OK
	} else {
		reply.Status = airlineproto.EFLIGHTEXISTS
	}
	
	return nil
}

func (as *AirlineServer) GetFlight(args *airlineproto.GetArgs, reply *airlineproto.GetReply) error {
	// send request to store handler
	cmdArgs, _ := json.Marshal(*args)
	paxosCmd := fmt.Sprintf("%s@GF@%s", as.myhostport, cmdArgs)
	
	req := &reqContent{PAXOS_REQUEST, paxosCmd, nil}
	replyc := make(chan interface{})
	as.reqChan <- &Request{req, replyc, true}
	
	// process answer
	info := <- replyc
	if info != nil {
		// get answer
		reply.Status = airlineproto.OK
		reply.FlightInfo = *(info.(*airlineproto.FlightInfo))
	} else {
		reply.Status = airlineproto.EFLIGHTNOTFOUND
	}
	return nil
}

func (as *AirlineServer) GetAllFlightIDs(args *airlineproto.GetAllArgs, reply *airlineproto.GetAllReply) error {
	// send request to store handler
	cmdArgs, _ := json.Marshal(*args)
	paxosCmd := fmt.Sprintf("%s@GA@%s", as.myhostport, cmdArgs)
	
	req := &reqContent{PAXOS_REQUEST, paxosCmd, nil}
	replyc := make(chan interface{})
	as.reqChan <- &Request{req, replyc, true}
	
	// process answer
	list := (<- replyc).([]string)
	
	reply.FlightIDs = list
	reply.Status = airlineproto.OK
	return nil
}


func (as *AirlineServer) Transaction(args *airlineproto.TranArgs, reply *airlineproto.TranReply) error {
	// send request to store handler
	cmdArgs, _ := json.Marshal(*args)
	paxosCmd := fmt.Sprintf("%s@TA@%s", as.myhostport, cmdArgs)
	req := &reqContent{PAXOS_REQUEST, paxosCmd, nil}
	
	// wait for response
	switch args.TranType {
	case transaction.TRANS_INIT, transaction.TRANS_RESPONSE, transaction.TRANS_OLD_RESPONSE:
		as.reqChan <- &Request{req, nil, true}
		reply.Status = airlineproto.OK
	case transaction.TRANS_OLD:
		replyc := make(chan interface{})
		as.reqChan <- &Request{req, replyc, true}
		decision := (<-replyc).(int)
		reply.Status = decision
	}
	
	return nil
	
}

// Get command from Paxos
func (as *AirlineServer) PaxosResponse(cmd string/*, own bool*/) {
	cmdArgs := strings.Split(cmd, "@")
	cmdNO, _ := strconv.Atoi(cmdArgs[0])
	
	hostport := cmdArgs[1]
	op := cmdArgs[2]
	argString := []byte(cmdArgs[3])
	// paxos should tell me "own"!!
	own := false
	if hostport == as.myhostport {
		own = true
	}
	switch op {
	case "CF":
		var args airlineproto.PutArgs
		json.Unmarshal(argString, &args)
		
		var replyc chan interface{}
		var exist bool
		if own == true {
			replyc, exist = as.oldReplycMap[cmdNO]
			if exist == false {
				return;
			}
			
			delete(as.oldReplycMap, cmdNO)
		}
		as.PaxosCreateFlight(&args, replyc, own)
		
	case "GF":
		var args airlineproto.GetArgs
		json.Unmarshal(argString, &args)
		
		var replyc chan interface{}
		var exist bool
		if own == true {
			replyc, exist = as.oldReplycMap[cmdNO]
			if exist == false {
				return;
			}
			
			delete(as.oldReplycMap, cmdNO)
		}
		as.PaxosGetFlight(&args, replyc, own)
		
	case "GA":
		var args airlineproto.GetAllArgs
		json.Unmarshal(argString, &args)
		
		var replyc chan interface{}
		var exist bool
		if own == true {
			replyc, exist = as.oldReplycMap[cmdNO]
			if exist == false {
				return;
			}
			delete(as.oldReplycMap, cmdNO)
		}
		as.PaxosGetAllFlightIDs(&args, replyc, own)
		
	case "TA":
		var args airlineproto.TranArgs
		json.Unmarshal(argString, &args)
		
		var replyc chan interface{}
		var exist bool
		if own == true {
			replyc, exist = as.oldReplycMap[cmdNO]
			if exist == false {
				return;
			}
			delete(as.oldReplycMap, cmdNO)
		}
		as.PaxosTransaction(&args, replyc, own)
	}
}




