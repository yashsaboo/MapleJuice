# MapleJuice
Scratch implementation of distributed computational framework which features protocols like group membership, leader election, and failure recovery, distributed file system and map-reduce algorithm for multiple applications ranging from simple word count to recommending friends based on social network graph. For testing, distributed grep was built to query terabytes of log data distributed over several machines.

This was done as a coursework project for CS425 Distributed Systems, which I took as a grad student in Fall 19 at Univeristy of Illinois at Urbana-Champaign.

## File Details

Codebase includes a Golang project which is distributed into 4 checkpoints:

1. MP1: Distributed Grep
2. MP2: Distributed Group Membership
3. MP3: Simple Distributed File System
4. MP4: MapleJuice
5. Hadoop Scripts: These python scripts ran on Hadoop system to compare the performance of MapleJuice with Hadoop's Mapreduce.

Main codebase for distributed framework comparable to simple Hadoop system is in /MapleJuice(MP4) folder. Other folders are used to build the final one, which has its unit testing reports included in it. Please read the report to know more about the implementation.

## Requirements to run the code
Golang

### Contributor: [Yash Saboo](https://github.com/yashsaboo)
