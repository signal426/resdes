package soldr

type TraitType uint32

const (
	Unspecified TraitType = iota
	NoTraits
	NotZero
	NotEqual
	Custom
)

var _ SubjectTrait = (*trait)(nil)

// trait is a feature of a subject
type trait struct {
	// the trait type
	traitType TraitType
	// the trait that this trait is composed with
	andTrait *trait
	// the trait that this trait is composed with
	orTrait *trait
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

func (t *trait) And() SubjectTrait {
	return t.andTrait
}

func (t trait) Type() TraitType {
	return t.traitType
}

func (t *trait) Or() SubjectTrait {
	return t.orTrait
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
