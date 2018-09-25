package lrmp

type errorEvent struct {
	source  Entity
	loser   Entity
	cause   int
	seqlost int
}

const (
	Unknown = 0

	/**
	 * The error cause: out of buffer error, i.e., no enough buffer space.
	 */
	BufferOverrun = 1

	/**
	 * The error cause: maximum number of repair requests reached.
	 */
	MaxTriesReached = 2

	/**
	 * The error cause: the sender is lost.
	 */
	SenderLost = 3

	/**
	 * The error cause: the sender is gone.
	 */
	SenderGone = 4
)

func newErrorEvent() *errorEvent {
	e := errorEvent{}
	return &e
}
