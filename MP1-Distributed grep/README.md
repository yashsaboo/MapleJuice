# RPC with Regex on Log Files

## Prerequisite
go, version 1.11.5; grep (GNU grep), version 2.20

## How to run the code
### Overall Process
#### For Distributed Grepping Log Files
1. Place the log files on each VM according to the VM numbers
2. Start the servers (server.go) on each VM specifying port 5566 and path to log file on each VM.
3. Run the client (client.go) program on any of the VMs while specifying the target regex pattern.
4. Client displays the results and done.

#### For Running Unit Test
1. Place the log files on each VM according to the VM numbers. Place a copy of all log files in the same directory as the unitTesting program.
2. Start the servers specifying port 5566 and path to log file on each VM.
3. Run the unit testing program (unitTest.go) while specifying the log file name pattern (see below section unitTest.go for more information).
4. Check if the tests in the unit testing program passed or not.

### Server (server.go)
The Server in this architecture is the daemon-like program living on each VM. The server(s) will serve RPC calls from the Client and return the regex-matching log entries back to the Client.
To run ```server.go```, open a terminal on a given VM and type the following command to start the server:
```
go run server.go <insert-port-number> <insert-log-filepath>
```
with ```<insert-port-number>``` being the port that the server will listen on for RPC calls and ```<insert-log-filepath>``` being the path to the log file that the server will serve. __Note that due to the IP addresses and port numbers being hard coded in ```client.go```, one would like to use 5566 as the port number for the server(s)__.

For instance,
```
go run server.go 5566 dummyFileForRegex.txt
```

### Client (client.go)
The Client in this architecture is the remote querying program that performs RPC on the server and retrieve the regex-matching entries from the log files of each server, after which combining them together for display.
To run ```client.go```, open a new terminal on a VM and type the following command to run the client:
```
go run client.go <regex-pattern>
```
with ```<regex-pattern>``` being the pattern that one would pass to the GNU ```grep``` with ```-E``` option on to match file contents.

For instance,
```
go run client.go reg
```

### genlog.go
```genlog.go``` is the program that generates the needed logs to do the unit testing part of this MP. The generate log program support two different generation schemes, with the first being based on total line count of the target log file and the other being based on the target file size.

To generate log in the total line count scheme, open a terminal on a given VM and type the following command to generate log file
```
go run genlog.go <vm-number> <log-file-path> <line-count-of-log>
```
with ```<vm-number>```being the number of the VM at which the generated log should be placed, ```<log-file-path>``` being the path to store the generated log file and ```<line-count-of-log>``` being a positive integer specifying the target line count.

For instance,
```
go run genlog.go 1 machine.1.log 10000
```
Log files generated in this manner will have the exact line count as specified in the command argument, with some of the entries being special entries that will only exist in log files of one(VM 3)/some(odd-numbered VMs)/all VMs.

To generate log in the target file size scheme, open a terminal on a given VM and type the following command to generate log file
```
go run genlog.go <vm-number> <log-file-path> -1 <file-size>
```
with ```<vm-number>``` being the number of the the VM at which the generated log should be placed, ```<log-file-path>``` being the path to store the generated log file and ```<file-size>``` being a positive integer which specifies the target file size.
For instance,
```
go run genlog.go 1 machine.1.log -1 6000000
```
Log files generated in this manner will have __roughly__ the file size specified in the command argument. __No__ special entries will appear in log files generated this way.

### unitTest.go
```unitTest.go``` is responsible to run unit testing.

To unit test, make sure the servers are running, along with a client. Also, on the client machine, one should have all the log files, named ```logFileName<insert VM number>.log```, of all the VMs where the servers are running. To execute the unit testing, type the following command
```
go run unitTest.go <log-file-name-without-VM-number>
```

for instance,
```
go run unitTest.go logFile
```
where and "logFile" is the name of the Log files present on the servers such as "logFile1.log" (for VM1), "logFile2.log" (for VM2) and so on.
The architecture of the unit testing is shown in the picture below:
![Unit Testing Architecture](https://i.imgur.com/laIdx7u.jpg)
