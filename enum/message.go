package enum

type MessageType string

const (
	MessageTypeBinary MessageType = "binary"
	MessageTypeText   MessageType = "text"
)

func (m MessageType) String() string {
	return string(m)
}
