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
	handleTimerTask(data interface{}, time time.Time)
}

type timerManager struct {
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

func newTimerManager() *timerManager {
	em := &timerManager{wakeup: make(chan bool, 16)}

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

			select {
			case <-em.wakeup:
			case <-time.After(timeout):
			}
			em.Lock()
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
				task.handler.handleTimerTask(task.data, task.time)
			}
		}
	}()

	return em
}

func (em *timerManager) recallTimer(task *timerTask) {
	em.Lock()
	defer em.Unlock()
	for next := em.tasks.Front(); next != nil; next = next.Next() {
		if next.Value == task {
			em.tasks.Remove(next)
			return
		}
	}
}
func (em *timerManager) registerTimer(ms int, handler timerHandler, data interface{}) *timerTask {
	t := timerTask{time: addMillis(time.Now(), ms), handler: handler, data: data}
	em.Lock()

	// insert in ascending order
	for elem := em.tasks.Front(); ; elem = elem.Next() {
		if elem == nil {
			em.tasks.PushFront(&t)
			break
		} else if t.time.Before(elem.Value.(*timerTask).time) {
			em.tasks.InsertBefore(&t, elem)
			break
		}
	}

	em.Unlock()

	em.wakeup <- true

	return &t
}
