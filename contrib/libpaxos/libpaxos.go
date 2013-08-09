package libpaxos

import (
	"log"
	"net/rpc"
	"P3-f12/official/lsplog"
	"time"
	"strings"
	"strconv"
)

//Message Types
const (
	NOP = 0
	PREPARE = 1
	PREPARE_OK = 2
	PREPARE_REJECT = 3
	ACCEPT = 4
	ACCEPT_OK = 5
	ACCEPT_REJECT = 6
	COMMIT = 7
	//The following are not received via RPC
	PROPOSE = 8
	LEARN = 9
	EPOCH = 10
)

//Paxos protocol messages
type Msg struct {
	MsgType int
	Proposal int
	Value string
	Instance int
}

//RPC messages to libpaxos
type Packet struct {
	PacketMsg Msg
	PacketFrom string //Address of replica
}

type Reply struct {
}

type Libpaxos struct {
	replicas []string //Addresses of all replicas
	self string //Own address
	
	c *Coordinator
	l *Learner
	a *Acceptor
	
	highestLearnt int //Highest instance number for which value was commited
	lastCommandExecuted int //Instance number which was last executed at paxos client
	
	pc PaxosClient
	
	operateReqC chan *Packet 
	
	unsentValues map[string] *Packet
	executeHistory map[string] bool
	
	proposeCounter int //Unique identifier so that same propose is not executed more than once
	
	nodepos int
}

func NewLibpaxos(self string, replicas []string, q int, nodepos int, pc PaxosClient) *Libpaxos {
	lsplog.SetVerbose(4)
	
	lp := &Libpaxos{}
	lp.nodepos = nodepos
	
	numnodes := len(replicas)
	lp.self = self
	lp.replicas = replicas
	
	lp.highestLearnt = 0
	
	//Initialize all 'roles'
	lp.c = &Coordinator{nodepos: nodepos, nodes: numnodes, quorum: q}
	lp.c.init()
	
	lp.a = &Acceptor{}
	lp.a.init()
	
	lp.l = &Learner{quorum: q}
	lp.l.init()
	
	lp.pc = pc
	lp.lastCommandExecuted = 0
	//rpc.Register(lp) 
	
	lp.proposeCounter = 0
	lp.unsentValues = make(map[string] *Packet)
	lp.executeHistory = make(map[string] bool)
	
	lp.operateReqC = make(chan *Packet)
	go lp.operateHandler()
	
	//Epoch
	go func() {
		for {
			d := time.Duration(1000) * time.Millisecond
			if d.Nanoseconds() > 0 {
				time.Sleep(d)
			}
			
			p := &Packet{}
			p.PacketFrom = lp.self
			p.PacketMsg = Msg{}
			p.PacketMsg.MsgType = EPOCH
			p.PacketMsg.Value = ""
			
			lp.operateReqC <- p
		}
	}()
	
	return lp
}

func (lp *Libpaxos) operateHandler() {
	for {
		// get new request
		//lsplog.Vlogf(6, "[Libpaxos] here1 ")
		p := <- lp.operateReqC
		//lsplog.Vlogf(6, "[Libpaxos] Processing packet from: %s", p.PacketFrom)
		lp.handleOperate(p)
	}
}

func (lp *Libpaxos) handleOperate(p *Packet) {

	lsplog.Vlogf(6, "[Libpaxos] Message type %d. Received packet from %s", p.PacketMsg.MsgType, p.PacketFrom)
	
	switch p.PacketMsg.MsgType {
	case PROPOSE:
		lp.unsentValues[p.PacketMsg.Value] = p 
	case EPOCH:
		for _, p := range lp.unsentValues {
			lsplog.Vlogf(6, "[Libpaxos] Retry unsent value ", p.PacketMsg.Value)
			lp.operate(p)
		}
	}
	
	//Pass message to all roles and broadcast a message if required
	//Each role processes mutually exclusive message types => no switch required
	PacketMsg := lp.c.operate(p)
	lp.broadcast(PacketMsg)
	
	PacketMsg, learnt, learntValue := lp.l.operate(p)
	lp.broadcast(PacketMsg)
	
	if learnt > 0 {
		if learnt > lp.highestLearnt {
			lp.highestLearnt = learnt	
			lp.c.instance = lp.highestLearnt	
		}
		
		//See if commmand(s) can be executed
		history := lp.l.getHistory()
		
		//Print history
		//printHistory(history)
		
		if(len(history) == lp.highestLearnt) {
			lsplog.Vlogf(6, "[Libpaxos] Execute Command(s)")
			
//			if lp.highestLearnt >= 60 {
//				lp.printHistory(history)
//			}
			
			//History is up to date. Execute command
			for i := lp.lastCommandExecuted+1; i<=lp.highestLearnt; i+=1 {
				v := history[i]
				lp.execute(v)
			}
			lp.lastCommandExecuted = lp.highestLearnt
		} else {
			lsplog.Vlogf(6, "[Libpaxos] Recover to:%d", lp.highestLearnt);
			//Recover missing values
			for i := 1; i< lp.highestLearnt ; i+=1 {
				_, f := history[i]
				if !f {
					lp.learn(i)
				}
			}
		}
		
		//Remove value from unsent list
		_, f := lp.unsentValues[learntValue]
		if f {
			delete(lp.unsentValues, learntValue)
		}
	}
	
	PacketMsg = lp.a.operate(p)
	lp.broadcast(PacketMsg)
}

func (lp *Libpaxos) printHistory(history map[int] string) {
		log.Printf("[Libpaxos %s] History", lp.self);
		
		for i := 1; i<=len(history); i+=1 {
			v := history[i]
			log.Printf(" %d:%s", i, lp.getFromModifiedValue(v))
		}
}

func (lp *Libpaxos) ProposeCommand(value string, reply *Reply) error {
	p := &Packet{}
	p.PacketFrom = lp.self
	p.PacketMsg = Msg{}
	p.PacketMsg.MsgType = PROPOSE
	p.PacketMsg.Value = lp.getModifiedValue(value)
	
	lp.proposeCounter += 1
	lp.operate(p)
	return nil
}

func (lp *Libpaxos) getModifiedValue(value string) string {
	return strconv.Itoa(lp.nodepos)+"$$"+strconv.Itoa(lp.proposeCounter)+"$$"+value
}

func (lp *Libpaxos) getFromModifiedValue(mvalue string) string {
	split := strings.Split(mvalue, "$$")
	return split[2]
}

func (lp *Libpaxos) operate(p *Packet) {
	//TODO:
	//lsplog.Vlogf(6, "[Libpaxos] Operate on packet from %s", p.PacketFrom)
	go func() {
		lp.operateReqC <- p
	//lp.handleOperate(p)
	} ()
}


func (lp *Libpaxos) learn(instance int) {
	p := &Packet{}
	p.PacketFrom = lp.self
	p.PacketMsg = Msg{}
	p.PacketMsg.MsgType = LEARN
	p.PacketMsg.Instance = instance
	
	lp.operate(p)
}

func (lp *Libpaxos) execute(value string) {
	_, f := lp.executeHistory[value]
	if !f {
		lp.executeHistory[value] = true
		lp.pc.PaxosResponse(lp.getFromModifiedValue(value))
	}
}

//RPC
func (lp *Libpaxos) ReceiveMessage(p Packet, reply *Reply) error {
	//lsplog.Vlogf(6, "[Libpaxos] Receive PacketFrom: %s", p.PacketFrom)
	lp.operate(&p)
	return nil
}

//Broadcast message to all replicas
func (lp *Libpaxos) broadcast(PacketMsg *Msg) {
	if PacketMsg != nil {
		//lsplog.Vlogf(6, "[Libpaxos] Broadcast type: %d", PacketMsg.MsgType)
			
		p := Packet{}
		p.PacketFrom = lp.self
		p.PacketMsg = *PacketMsg
		
		var reply Reply
		for _, r := range lp.replicas {
//			if r == lp.self {
//				continue
//			}
			
			//lsplog.Vlogf(6, "[Libpaxos] Broadcast: %s", r)
			
			client, err := rpc.DialHTTP("tcp", r)
			if lsplog.CheckReport(6, err) {
				lsplog.Vlogf(6, "[Libpaxos] Broadcast to %s failed", r)
				//client.Close()
				continue
			}
			
			err = client.Call("Libpaxos.ReceiveMessage", p, &reply)
			if lsplog.CheckReport(1, err) {
				lsplog.Vlogf(6, "[Libpaxos] Broadcast call to %s failed", r)
			}
			client.Close()
		}
	}
}