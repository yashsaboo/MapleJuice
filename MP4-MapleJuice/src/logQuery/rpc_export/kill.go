package rpc_export

import (
	"errors"
)

func Kill(args *KillCommand, reply *KillResponse) error {

	UL.Printf("Kill Request Received!")

	if kill_signal == nil {
		UL.Printf("Kill Channel Was Nil! Cannot Send Kill Signal")
		return errors.New("No Kill Channel Detected")
	}

	UL.Printf("Sending Kill Signal to Main Node Thread")

	*kill_signal <- true
	return nil
}
