package rpc_export

import (
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Executes a Log Search. Massive security implications here since we just run a slice of strings as arguments to a remote exec call. But this is code for class.
func (*Node) LogSearch(args *LogSearchQuery, reply *LogSearchReply) error {

	// Log the search
	UL.Printf("Log Search %s", args.GrepArgs)

	log_file_name := fmt.Sprintf(args.Location, GrepID)

	command := exec.Command("bash", "-c", "grep "+args.GrepArgs+" "+log_file_name)

	fmt.Println("CMD: " + "grep " + args.GrepArgs + " " + log_file_name)

	//Run this user-supplied command
	result, err := command.CombinedOutput()
	if err != nil {
		if len(result) > 1 {
			UL.Printf("Error running command: %s\n", err, "STDERR: ", result)
			return errors.New("Command returned an error: <<" + string(result[:len(result)-2]) + ">>")
		} else {
			// Send an empty reply if result was empty (just null terminator)
			reply.GrepID = GrepID
			reply.Logs = []string{}
			return nil
		}
	}

	// Populate the reply
	reply.Logs = strings.Split(string(result), "\n")
	reply.GrepID = GrepID
	return nil
}
