package libpaxos

import (
	//"log"
	"P3-f12/official/lsplog"
)

//Info for single instance
type AcceptorProtocol struct {
	proposal int //Highest proposal number seen
	value	string //Value accepted
	proposalAccepted int //Proposal number accepted
}

func (ap *AcceptorProtocol) init() {
	ap.proposal = 0
	ap.value = ""
	ap.proposalAccepted = 0
}

func (ac *Acceptor) init() {
	ac.proto = make(map[int] *AcceptorProtocol)
}

//Libpaxos acceptor object
type Acceptor struct {
	proto map[int] *AcceptorProtocol
}

func (ac *Acceptor) operate (p *Packet) *Msg {
	msg := p.PacketMsg
	proto, f := ac.proto[msg.Instance]
	if !f {
		//Create protocol info for new instance
		ap := &AcceptorProtocol{}
		ap.init()
		
		ac.proto[msg.Instance] = ap
		proto = ap
	}
	
	switch (msg.MsgType) {
	//Phase 1: Prepare
	case PREPARE:
		lsplog.Vlogf(6, "[Acceptor] PREPARE received for instance %d.", msg.Instance)
		if msg.Proposal > proto.proposal {
			proto.proposal = msg.Proposal
			return &Msg{ PREPARE_OK, proto.proposalAccepted, proto.value, msg.Instance}
		} else {
			lsplog.Vlogf(6, "[Acceptor] Reject proposal number %d.", msg.Proposal)
			return &Msg{ PREPARE_REJECT, proto.proposal, "", msg.Instance}
		}
		
	//Phase 2: Accept
	case ACCEPT:
		lsplog.Vlogf(6, "[Acceptor] ACCEPT received for instance %d.", msg.Instance)
		lsplog.Vlogf(6, "[Acceptor] n:%d n_h:%d n_a:%d", msg.Proposal, proto.proposal, proto.proposalAccepted)
		if msg.Proposal >= proto.proposal && msg.Proposal != proto.proposalAccepted {
			proto.proposal = msg.Proposal
			proto.proposalAccepted = msg.Proposal
			proto.value = msg.Value
			
			return &Msg{ ACCEPT_OK, msg.Proposal, msg.Value, msg.Instance}
		} else if msg.Proposal < proto.proposal {
			//Seen a higher proposal number
			lsplog.Vlogf(6, "[Acceptor] Reject proposal number %d.", msg.Proposal)
			return &Msg{ ACCEPT_REJECT, 0, "", 0}
		}
	}
	return nil
}