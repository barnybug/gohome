package gorfxtrx

// Struct for the Reset packet type.
type Reset struct {
}

func (self *Reset) Send() []byte {
	return []byte{0x0d, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
}

// Reset packet constructor.
func NewReset() (*Reset, error) {
	return &Reset{}, nil
}
