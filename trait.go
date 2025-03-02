package soldr

import "fmt"

type TraitType uint32

const (
	Unspecified TraitType = iota
	NoTraits
	NotZero
	NotEqual
	Custom
)

// trait is a feature of a subject
type trait struct {
	// the trait type
	traitType TraitType
	// the trait that this trait is composed with
	andTrait SubjectTrait
	// the trait that this trait is composed with
	orTrait SubjectTrait
	// the value that the subject cannot have
	notEqualTo interface{}
}

func Traitless() *trait {
	return &trait{
		traitType: NoTraits,
	}
}

func NotZeroTrait() *trait {
	return &trait{
		traitType: NotZero,
	}
}

func NotEqualTrait(ne interface{}) (*trait, error) {
	if !validValue(ne) {
		return nil, fmt.Errorf("cannot compare non-primitive types")
	}
	return &trait{
		traitType:  NotEqual,
		notEqualTo: ne,
	}, nil
}

func validValue(value interface{}) bool {
	switch value.(type) {
	case bool, string,
		int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, uintptr,
		float32, float64,
		complex64, complex128:
		return true
	default:
		return false
	}
}

// reports whether or not a trait can be combined
// with other traits into a chain. Undefined or "No Traits"
// trait types cannot be combined with and will ignore any calls
// to do so.
func IsCompoundTrait(t TraitType) bool {
	return t != Unspecified && t != NoTraits
}

func (t *trait) and(and *trait) *trait {
	if !IsCompoundTrait(t.traitType) {
		return t
	}
	t.andTrait = and
	return t
}

func (t *trait) NotEqualTo() interface{} {
	return t.notEqualTo
}

// And implements SubjectTrait.
func (t *trait) And(st SubjectTrait) {
	t.andTrait = st
}

func (t *trait) GetAnd() SubjectTrait {
	return t.andTrait
}

// Or implements SubjectTrait.
func (t *trait) Or(st SubjectTrait) {
	t.orTrait = st
}

func (t *trait) GetOr() SubjectTrait {
	return t.orTrait
}

func (t trait) Type() TraitType {
	return t.traitType
}

func (t *trait) Valid() bool {
	return t != nil
}

func (t *trait) or(or *trait) *trait {
	if !IsCompoundTrait(t.traitType) {
		return t
	}
	t.orTrait = or
	return t
}

func (t *trait) FaultString() string {
	if t.Type() == NotZero {
		return "it should not be zero"
	}
	return "it should not be equal to the supplied value"
}
