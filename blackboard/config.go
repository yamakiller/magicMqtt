package blackboard

//Config 系统配置信息
type Config struct {
	WorkGroupID      int64 `yaml:"workGroup" json:"workGroup"`
	WorkID           int64 `yaml:"work" json:"work"`
	Keepalive        int   `yaml:"keepAlive" json:"keepAlive"`
	OfflineQueueSize int   `yaml:"offlineQueueSize" json:"offlineQueueSize"`
	MessageQueueSize int   `yaml:"messageQueueSize" json:"messageQueueSize"`
	MessageSize      int   `yaml:"messageSize" json:"messageSize"`
	BufferSize       int   `yaml:"bufferSize" json:"bufferSize"`
}
