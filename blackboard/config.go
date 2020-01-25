package blackboard

type Config struct {
	WorkGroupID      int64
	WorkID           int64
	Keepalive        int
	MessageQueueSize int
	MessageSize      int
	BufferSize       int
}
