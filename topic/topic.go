package topic

func NewTopic(name string) *Topic {
	return &Topic{_name: name}
}

type Topic struct {
	_name string
}

func (slf *Topic) Name() string {
	return slf._name
}
