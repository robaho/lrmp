package lrmp

type eventManager struct {
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
	return &eventManager{}
}
