package lrmp

import (
	"container/list"
	"sync"
	"time"
)

type timerTask struct {
	time    time.Time
	data    interface{}
	handler timerHandler
}

type timerHandler interface {
	handleTimerEvent(data interface{}, time time.Time)
}

type eventManager struct {
	sync.Mutex
	tasks  list.List
	wakeup chan bool
}

/**
 * the event type: unrecoverable reception error. This event is generated
 * when a part of data is missing in the received data stream, generally
 * due to serious network problems.
 */
const UNRECOVERABLE_SEQUENCE_ERROR = 1

/**
 * the event type: end of sequence. This event is generated when a data sender
 * is lost or gone. It allows upper layer to clean-up incomplete data object.
 */
const END_OF_SEQUENCE = 2

func newEventManager() *eventManager {
	em := &eventManager{wakeup: make(chan bool, 16)}

	go func() {
		for {
			em.Lock()

			timeout := time.Duration(10 * time.Second)

			if em.tasks.Len() > 0 {
				first := em.tasks.Front().Value.(*timerTask)
				timeout = first.time.Sub(time.Now())
			}

			if timeout < 0 {
				timeout = 1
			}

			em.Unlock()

			if isDebug() {
				logDebug("timer timeout is ", timeout)
			}

			select {
			case <-em.wakeup:
			case <-time.After(timeout):
			}
			em.Lock()
			if isDebug() {
				logDebug("checking for timer tasks")
			}
			var task *timerTask
			if em.tasks.Len() > 0 {
				task = em.tasks.Front().Value.(*timerTask)
				if task.time.Before(time.Now()) {
					em.tasks.Remove(em.tasks.Front())
				} else {
					task = nil
				}
			}

			em.Unlock() // need to unlock because task handler might try to submit another task...

			if task != nil {
				if isDebug() {
					logDebug("firing timer task", task)
				}
				task.handler.handleTimerEvent(task.data, task.time)
				if isDebug() {
					logDebug("fired timer task", task)
				}
			}
		}
	}()

	return em
}

func (em *eventManager) recallTimer(ev *timerTask) {
	em.Lock()
	defer em.Unlock()
	for next := em.tasks.Front(); next != nil; next = next.Next() {
		if next.Value == ev {
			em.tasks.Remove(next)
			return
		}
	}
}
func (em *eventManager) registerTimer(ms int, handler timerHandler, data interface{}) *timerTask {
	t := timerTask{time: addMillis(time.Now(), ms), handler: handler, data: data}
	em.Lock()

	// insert in ascending order
	for next := em.tasks.Front(); ; next = next.Next() {
		if next == nil || t.time.Before(next.Value.(*timerTask).time) {
			if next == nil {
				em.tasks.PushFront(&t)
			} else {
				em.tasks.InsertBefore(&t, next)
			}
			break
		}
	}

	if isDebug() {
		logDebug("scheduled timer", &t, " in ", t.time.Sub(time.Now()))
	}

	em.Unlock()

	em.wakeup <- true

	return &t
}
