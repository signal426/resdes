package soldr

type Policy uint32

const (
	NonZero Policy = iota
	NotEqualTo
	MustEqual
)

type Condition uint32

const (
	Always Condition = iota
	InMask
)
