package libpaxos

import (
	//"log"
	"P3-f12/official/lsplog"
)

//Information for single instance
type LearnerProtocol struct{
	responses map[string] bool //Has replica voted before?
	valueCounts map[string] int //Number of voted received for different values
	proposal int //Highest proposal number seen
	value string
	commit bool//Has the value been commited
}

func (lp *LearnerProtocol) init() {
	lp.responses = make(map[string] bool)
	lp.valueCounts = make(map[string] int)
	lp.proposal = 0
	lp.value = ""
	lp.commit = false
}

//Libpaxos learner object
type Learner struct {
	quorum int
	proto map[int] *LearnerProtocol
}

func (lr *Learner) init() {
	lr.proto = make(map[int] *LearnerProtocol)
}

func (lr *Learner) getHistory() map[int] string {
	h := make(map[int] string)
	for i, p := range lr.proto {
		if p.commit == true {
			h[i] = p.value
		}
	}
	
	return h
}

func (lr *Learner) operate (p *Packet) (*Msg, int, string) {
	msg := p.PacketMsg
	proto, f := lr.proto[msg.Instance]
	if !f {
		//Create protocol info for new instance
		lp := &LearnerProtocol{}
		lp.init()
		
		lr.proto[msg.Instance] = lp
		proto = lp
	}
	
	//Value has already been commited for this instance
	if proto.commit {
		lsplog.Vlogf(6, "[Learner] Value already commited for instance %d", msg.Instance)
		
		//Aid recovery
		if msg.MsgType == NOP {
			lsplog.Vlogf(6, "[Learner] Aiding recovery for instance %d. Sent value %s", msg.Instance, proto.value)
			return &Msg{COMMIT, proto.proposal, proto.value, msg.Instance}, 0, ""
		}
		return nil, 0, ""
	}
	
	switch (msg.MsgType) {
	//Phase 2: Does majority agree?
	case ACCEPT_OK:
		lsplog.Vlogf(6, "[Learner] ACCEPT_OK received for instance %d. n:%d n_h: %d", msg.Instance, msg.Proposal, proto.proposal)
		if msg.Proposal >= proto.proposal {
			if msg.Proposal > proto.proposal {
				proto.proposal = msg.Proposal
				proto.responses = make(map[string]bool)
				proto.valueCounts = make(map[string]int)
			}
			
			_, f := proto.responses[p.PacketFrom]
			if !f {
				proto.valueCounts[msg.Value]++
				proto.responses[p.PacketFrom] = true
				
				if proto.valueCounts[msg.Value] >= lr.quorum {
					proto.commit = true
					proto.value = msg.Value
					
					lsplog.Vlogf(6, "[Learner] Learnt value %s for instance %d.", msg.Value, msg.Instance)
					//Commit value
					return &Msg{COMMIT, msg.Proposal, msg.Value, msg.Instance}, msg.Instance, msg.Value
				}
			} else {
				lsplog.Vlogf(6, "[Learner] Duplicate vote")
			}
		}
	//TODO: ACCEPT_REJECT
	
	//Phase 3: Quorum has been reached. Value learnt.
	case COMMIT:
		proto.commit = true
		proto.value = msg.Value
					
		lsplog.Vlogf(6, "[Learner] COMMIT value:%s instance:%d.", msg.Value, msg.Instance)
		return nil, msg.Instance, msg.Value
	}
	
	return nil, 0, ""
}
	