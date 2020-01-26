package blackboard

//Config 系统配置信息
type Config struct {
	WorkGroupID      int64
	WorkID           int64
	Keepalive        int
	MessageQueueSize int
	MessageSize      int
	BufferSize       int
}
