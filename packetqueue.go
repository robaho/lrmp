package lrmp

import (
	"container/list"
	"sync"
)

type packetQueue struct {
	sync.Mutex
	list.List
}

func (pq *packetQueue) isEmpty() bool {
	pq.Lock()
	defer pq.Unlock()
	return pq.Len() == 0
}

/**
 * adds the given packet to the queue.
 */
func (pq *packetQueue) enqueue(p *Packet) {
	pq.Lock()
	defer pq.Unlock()
	pq.PushFront(p)
}

func (pq *packetQueue) contains(p *Packet) bool {
	pq.Lock()
	defer pq.Unlock()
	for elem := pq.Front(); elem != nil; elem = elem.Next() {
		if elem.Value.(*Packet) == p {
			return true
		}
	}
	return false
}

func (pq *packetQueue) dequeue() *Packet {
	pq.Lock()
	defer pq.Unlock()
	front := pq.Front()
	if front == nil {
		return nil
	}
	p := front.Value.(*Packet)
	pq.Remove(front)
	return p
}

/**
* remove the packet with the given seqno from the queue.
 */
func (pq *packetQueue) remove(s *sender, seqno int64, scope int) {
	pq.Lock()
	defer pq.Unlock()
	for next := pq.Front(); next != nil; next = next.Next() {
		p := next.Value.(*Packet)
		if p.sender.getID() == s.id && p.seqno == seqno && p.scope == scope {
			pq.Remove(next)
			break
		}
	}
}

func (pq *packetQueue) cancel(s *sender, id int, scope int) {
	pq.Lock()
	defer pq.Unlock()
	for next := pq.Front(); next != nil; next = next.Next() {
		p := next.Value.(*Packet)
		if p.sender.getID() == s.id && p.retransmitID == id && p.scope == scope {
			pq.Remove(next)
			break
		}
	}
}
