package transaction

// transaction type
const (
	TRANS_INIT = iota
	TRANS_RESPONSE
	TRANS_OLD
	TRANS_OLD_RESPONSE
)

// transaction expirary time
const (
	EXPIRARY_SECONDS = 5
)

// response type
const (
	NOT_DECIDED = iota
	COMMIT
	ABORT
)
