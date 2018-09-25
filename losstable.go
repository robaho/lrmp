package lrmp

import "container/list"

type lossTable struct {
	list.List
}

func (lt *lossTable) clear() {
	lt.Init()
}

func (lt *lossTable) size() int {
	return lt.Len()
}

func (lt *lossTable) add(ev *lossEvent) {
	lt.PushFront(ev)
}

func (lt *lossTable) remove(ev *lossEvent) {
	for next := lt.Front(); next != nil; next = next.Next() {
		if ev == next.Value {
			lt.Remove(next)
			return
		}
	}
}

func (lt *lossTable) lookup(s *sender, reporter Entity) *lossEvent {
	for next := lt.Front(); next != nil; next = next.Next() {
		ev := next.Value.(*lossEvent)
		if ev.source == s && ev.reporter == reporter {
			return ev
		}
	}
	return nil
}
