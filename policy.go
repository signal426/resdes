package resdes

type Policy uint32

const (
	NonZero Policy = iota
	NotEqualTo
	MustEqual
	Custom
)

func (p Policy) String() string {
	switch p {
	case NonZero:
		return "non-zero"
	case MustEqual:
		return "must equal"
	case NotEqualTo:
		return "must not equal"
	case Custom:
		return "custom evaluation"
	default:
		return "unknown policy"
	}
}

type Condition uint32

const (
	Always Condition = iota
	InMask
)
