# CS425 MP2 README

### Overall Process
1. Start up the server process (introducer) on VM01, join the server process
2. Start up and join all the other server processes on other VMs
3. Join/Leave/Kill the server processes 
4. [Optional] Start up the MP1 programs as stated in MP1 README for remote log grepping
> Note that for the server on VM01 (introducer) to rejoin after leaving/failing, the server on VM02 must be alive and have joined the network
> Note that for the other server process to rejoin after leaving/failing, the server on VM01 (introducer) must be alive and have joined the network 

### Compile and Run the Code
To compile and run the server program, use the following command:
```
go run server.go HB_PORT DISSEM_PORT INTRO STORE_LOG_PATH [UDP_FAIL_RATE]
```
where ```HB_PORT``` is a port that the heartbeat thread listen on, ```DISSEM_PORT``` is a port the gossip thread listen on, ```INTRO``` is the flag to set to 1 if the server process is meant to be the introducer on VM01 and 0 otherwise, ```STORE_LOG_PATH``` is the path where the logs will be logged to, and the optional ```UDP_FAIL_RATE``` is the probabilty that a UDP message sent by the server process will be dropped, which is useful for the report measurement part.
For example, to run a server program acting as an introducer on VM01, run the following command on VM01:
```
go run server.go 5000 5001 1 log01.txt
```
To run a server program acting as a normal node, run the following command:
```
go run server.go 5000 5001 0 log.txt 0
```

### Commandline interface
When running the program, there are 4 available options that can be initiated by typing in 1,2,3,4 respectively. The options are namely

1. Show membership list - show the current membership list content (the server itself is not on the list)
2. Show self ID - print out the server ID
3. Join - join the peer network (note that the corresponding introducer has to be up and is currently joined for this to work)
4. Voluntarily leave - leave the peer network

The join operation requires that the server is currently not joined.
The show membership list operation, show self ID operation and leave operation requires that the server is currently joined in the network.