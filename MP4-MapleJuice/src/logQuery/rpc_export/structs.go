package rpc_export

// The RPC Query Object for Log Queries
// GrepArgs should be a list of arguments to grep like -n or -Po or "foo.*"
type LogSearchQuery struct {
	GrepArgs string
	Location string
}

// This is the reply to an RPC Log Query
type LogSearchReply struct {
	GrepID int
	Logs   []string
}

// Send one of these to rpc.Kill to kill a node
type KillCommand struct {
	K bool
}

// Reply to a kill command
type KillResponse struct {
	K bool
}
