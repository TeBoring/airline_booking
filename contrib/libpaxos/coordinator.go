package libpaxos

import (
	//"log"
	"P3-f12/official/lsplog"
)

//Protocol info for single instance
type CoordinatorProtocol struct {
	pvalue string //Value to propose
	aproposal int //highest n_a received
	avalue string //v_a corresponding to highest n_a received
	nvalue string //Value corresponding to highest n_a recieved Next value to propose
	responses map[string] bool //Maps replica address to vote
	proposal int //My proposal number
}

func (cp *CoordinatorProtocol) init() {
	cp.pvalue = ""
	cp.nvalue = ""
	cp.responses = make(map[string] bool)
	cp.proposal = 0
	cp.aproposal = 0
	cp.avalue = ""
}

//Libpaxos coordinator object
type Coordinator struct {
	nodepos int
	nodes int
	quorum int
	proto map[int] *CoordinatorProtocol
	instance int //highest instance number learnt
	nextproposal int
	highproposal int
}

func (c *Coordinator) init () {
	c.proto = make(map[int] *CoordinatorProtocol)
	c.instance = 0
	c.nextproposal = c.nodepos
}

//Monotonically increasing unique proposal number
func (c *Coordinator) newProposalNumber() int {
	p  := c.nextproposal
	c.nextproposal += c.nodes
	
	for c.nextproposal < c.highproposal {
		c.nextproposal += c.nodes
	}
	
	return p
}

func (c *Coordinator) newInstanceNumber() int {
	c.instance += 1
	return c.instance
}

func (c *Coordinator) operate (p *Packet) *Msg {
	msg := p.PacketMsg
	
	switch (msg.MsgType) {
	//Phase 1: I want to become the leader
	case PROPOSE:
		//proto, f := c.proto[msg.Instance]
		//Create protocol info for new instance
		cp := &CoordinatorProtocol{}
		cp.init()
		cp.pvalue = msg.Value
		cp.proposal = c.newProposalNumber()
		
		instance  := c.newInstanceNumber()
		c.proto[instance] = cp
		
		lsplog.Vlogf(6, "[Coordinator] Send PREPARE proposal number:%d, instance:%d, value:%s", 
			cp.proposal, instance, cp.pvalue)
		return &Msg{ PREPARE, cp.proposal, "", instance}
	
	//For recovery: learn value of missing instance
	case LEARN:	
		lsplog.Vlogf(6, "[Coordinator] Sending LEARN instance:%d", msg.Instance)
		return &Msg{ NOP, 0, "", msg.Instance}
		
	//Phase 2:
	case PREPARE_OK:
		proto, f := c.proto[msg.Instance]
		if f {
			lsplog.Vlogf(6, "[Coordinator] PREPARE_OK received, instance:%d", msg.Instance)
			//Send only once (additional prepare_ok should not trigger more messages)
			if proto.nvalue == "" {
				//Calculate highest n_a, v_a seen by acceptors
				if msg.Proposal > proto.aproposal {
					proto.aproposal = msg.Proposal
					proto.avalue = msg.Value
					
					lsplog.Vlogf(6, "[Coordinator] n_a: %d, v_a: %s", msg.Proposal, msg.Value)
				}
		
				//Record that replica has responded. Discard duplicate responses
				_, f := proto.responses[p.PacketFrom]
				if !f {
					proto.responses[p.PacketFrom] = true
				}
				
				//Propose value if majority accept me to be the leader
				if len(proto.responses) >= c.quorum {
					if proto.aproposal > 0 {
						proto.nvalue = proto.avalue
					} else {
						proto.nvalue = proto.pvalue
					}
					
					lsplog.Vlogf(6, "[Coordinator] Send ACCEPT n:%d instance:%d, value:%s", proto.proposal, msg.Instance, proto.nvalue)
					return &Msg{ ACCEPT, proto.proposal, proto.nvalue, msg.Instance}
				}
			}
		}
	case PREPARE_REJECT:
		c.highproposal = msg.Proposal
	}
	return nil
}