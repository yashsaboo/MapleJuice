package node

import (
	"time"
	"strconv"
	"strings"
)

// FlushMessageCache will remove all messages older than the max age from the cache
func (r *RecentMessageCache) FlushMessageCache(MaxAgeSeconds uint64) error {
	curTime := time.Now()
	millis := curTime.UnixNano() / 1000000
	for i := 0; i < len(r.RecentMessages); i++ {
		if r.RecentMessages[i].Timestamp+MaxAgeSeconds < uint64(millis*1000) {
			r.L.Lock()
			r.RecentMessages = append(r.RecentMessages[:i], r.RecentMessages[i+1:]...)
			r.L.Unlock()
		}
	}
	return nil
}

// Add will add a new message to the recent message cache
func (r *RecentMessageCache) Add(message string) error {
	curTime := time.Now()
	millis := curTime.UnixNano() / 1000000

	var newMsg RecentMessage
	messageFields := strings.Split(message, ",")

	newMsg.Type = messageFields[0]
	newMsg.OriginatorID, _ = strconv.ParseUint(messageFields[1], 10, 64)
	newMsg.Timestamp = uint64(millis)

	r.L.Lock()
	r.RecentMessages = append(r.RecentMessages, newMsg)
	r.L.Unlock()
	// fmt.Printf("Adding a new message to the cache with type %s and id %d\n", newMsg.Type, newMsg.OriginatorID)
	return nil
}

// Contains will check if a message is in the recent message cache
func (r *RecentMessageCache) Contains(message string) bool {
	var newMsg RecentMessage
	messageFields := strings.Split(message, ",")

	newMsg.Type = messageFields[0]
	newMsg.OriginatorID, _ = strconv.ParseUint(messageFields[1], 10, 64)

	for i := 0; i < len(r.RecentMessages); i++ {
		if (r.RecentMessages[i].OriginatorID == newMsg.OriginatorID) &&
			(r.RecentMessages[i].Type == newMsg.Type) {
			return true
		}
	}
	return false
}
