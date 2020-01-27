package node

// This file contains the data structures and helper functions for maintaining the neighbor and membership lists
import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// MembershipList struct with lock
type MembershipList struct {
	L       *sync.Mutex
	Members []OtherNode
}

// NeighborList struct with lock
type NeighborList struct {
	L         *sync.Mutex
	Neighbors []OtherNode
}

// Contains will check if a node is in the list. If it is, it will return the index. If it isn't, return -1
func (m *MembershipList) Contains(id uint64) int {
	for i := 0; i < len(m.Members); i++ {
		if m.Members[i].NodeID == id {
			return i
		}
	}
	return -1
}

// Add will add a new node to the list if it isn't already in the list
func (m *MembershipList) Add(node OtherNode) error {
	if m.Contains(node.NodeID) >= 0 {
		return nil
	}
	node.LastHeartbeat = uint64(time.Now().UnixNano() / 1000000) // Set the last heartbeat to give a small buffer
	m.L.Lock()
	m.Members = append(m.Members, node)
	m.L.Unlock()
	m.Sort()

	fmt.Printf("Added new member: (%d) New membership list:\n", node.NodeID)
	for i := 0; i < len(m.Members); i++ {
		fmt.Printf("Node %d (ID=%d)\n", m.Members[i].TCPPort-10000, m.Members[i].NodeID)
	}

	return nil
}

// Remove removes a node from the list if it exists
func (m *MembershipList) Remove(id uint64) error {
	idx := m.Contains(id)
	if idx < 0 {
		return errors.New("Tried to remove node that wasn't in membership list")
	}
	m.L.Lock()
	m.Members = append(m.Members[:idx], m.Members[idx+1:]...)
	m.L.Unlock()
	fmt.Printf("Removed a member: (%d) New membership list:\n", id)
	for i := 0; i < len(m.Members); i++ {
		fmt.Printf("Node %d (ID=%d)\n", m.Members[i].TCPPort-10000, m.Members[i].NodeID)
	}

	return nil
}

// Sort will sort the membership list by NodeID
func (m *MembershipList) Sort() {
	m.L.Lock()
	defer m.L.Unlock()
	sort.Slice(m.Members[:], func(i, j int) bool {
		return m.Members[i].NodeID < m.Members[j].NodeID
	})
}

// Update goes through the membership list and finds our new neighbors
func (n *NeighborList) Update(m *MembershipList, id uint64) error {
	m.Sort()

	previousNeighborList := []int{}
	for i := 0; i < len(n.Neighbors); i++ {
		previousNeighborList = append(previousNeighborList, n.Neighbors[i].TCPPort-10000)
	}

	myIndex := 0
	for myIndex = 0; myIndex < len(m.Members); myIndex++ {
		if m.Members[myIndex].NodeID == id {
			break
		}
	}

	if myIndex >= len(m.Members) {
		return errors.New("I am not in my own membership list")
	}
	if len(m.Members) == 0 {
		return errors.New("No members in membership list to update")
	}

	var newNeighbors []OtherNode // Create new, empty neighbor list
	curTime := uint64(time.Now().UnixNano() / 1000000)

	numMembers := len(m.Members)

	m.Members[(mod(myIndex+1, numMembers))].LastHeartbeat = curTime // Update the heartbeat so we don't get any false-positives
	m.Members[(mod(myIndex+2, numMembers))].LastHeartbeat = curTime
	m.Members[(mod(myIndex-1, numMembers))].LastHeartbeat = curTime
	m.Members[(mod(myIndex-2, numMembers))].LastHeartbeat = curTime

	newNeighbors = append(newNeighbors, m.Members[(mod(myIndex-2, numMembers))]) // Add the new neighbors in order
	newNeighbors = append(newNeighbors, m.Members[(mod(myIndex-1, numMembers))])
	newNeighbors = append(newNeighbors, m.Members[(mod(myIndex+1, numMembers))])
	newNeighbors = append(newNeighbors, m.Members[(mod(myIndex+2, numMembers))])

	n.Neighbors = newNeighbors

	// Report new neighbor list only if there were changes
	changeHappened := len(n.Neighbors) != len(previousNeighborList)
	for i := 0; i < len(n.Neighbors) && !changeHappened; i++ {
		if previousNeighborList[i] != (n.Neighbors[i].TCPPort - 10000) {
			changeHappened = true
		}
	}
	if changeHappened {
		fmt.Println("Updated neighbors: ")
		j := -2
		for i := 0; i < len(n.Neighbors); i++ {
			fmt.Printf("Neighbor %d is Node %d (ID=%d)\n", j, n.Neighbors[i].TCPPort-10000, n.Neighbors[i].NodeID)
			j++
			if j == 0 {
				j++
			}
		}
	}

	return nil
}

// mod performs the % (modulo) operation since Go does not have the implementation we want
func mod(a int, n int) int {
	if n == 0 {
		return 0
	}
	val := a - (n * int(a/n))
	if val < 0 {
		return val + n
	}
	return val
}
