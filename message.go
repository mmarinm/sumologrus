package sumologrus

import (
	"encoding/json"
)

func messageSize(m SumoLogicMesssage) int {
	var msgBytes []byte
	msgBytes, _ = json.Marshal(m)

	// The `+ 1` is for the comma that sits between each items of a JSON array.
	return len(msgBytes) + 1
}

type messageQueue struct {
	pending       []SumoLogicMesssage
	bytes         int
	maxBatchSize  int
	maxBatchBytes int
}

func (q *messageQueue) push(m SumoLogicMesssage) (b []SumoLogicMesssage) {
	messageSize := messageSize(m)
	if (q.bytes + messageSize) > q.maxBatchBytes {
		b = q.flush()
	}

	if q.pending == nil {
		q.pending = make([]SumoLogicMesssage, 0, q.maxBatchSize)
	}

	q.pending = append(q.pending, m)
	q.bytes += messageSize

	if b == nil && len(q.pending) == q.maxBatchSize {
		b = q.flush()
	}

	return
}

func (q *messageQueue) flush() (msgs []SumoLogicMesssage) {
	msgs, q.pending, q.bytes = q.pending, nil, 0
	return
}

const (
	maxBatchBytes = 1000000
)
