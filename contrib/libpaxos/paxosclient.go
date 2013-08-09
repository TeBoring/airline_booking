package libpaxos

import (

)

type PaxosClient interface {
	PaxosResponse( cmd string)
}