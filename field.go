package soldr

import (
	"errors"
	"reflect"
)

var (
	_                       PolicySubject = (*Field)(nil)
	ErrFieldNotSetInMessage error         = errors.New("expected field to be set in message but was not present")
	ErrFieldNotSetInMask    error         = errors.New("field not set in mask")
)

func isZero(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsZero()
}

func (f Field) MustBeSet() (string, bool) {
	if f.Zero() {
		return ErrFieldNotSetInMessage, false
	}
	return nil, true
}

func (f Field) MustBeSetIfInMask() (string, bool) {
	if !f.inMask {
		return nil, true
	}

	if f.Zero() && f.inMask {
		return ErrFieldNotSetInMessage, false
	}
	return nil, true
}

type Field struct {
	Path         string
	Value        interface{}
	inMask       bool
	setInMessage bool
}

func NewField(path string, value interface{}, inMask bool) *Field {
	return &Field{
		Path:   path,
		Value:  value,
		inMask: inMask,
	}
}

func (f Field) isComparable(i interface{}) bool {
	return reflect.TypeOf(f.Value) == reflect.TypeOf(i)
}

func (f Field) Zero() bool {
	return isZero(f.Value)
}

// CheckHasTraits implements PolicySubject.
func (f Field) HasTrait(t SubjectTrait) bool {
	switch t.Type() {
	case NotZero:
		return !f.Zero()
	case NotEqual:
		return f.Value != t.NotEqualTo()
	default:
		return true
	}
}

// ID implements PolicySubject.
func (f Field) ID() string {
	return f.Path
}
