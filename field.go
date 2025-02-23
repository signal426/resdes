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

func (f Field) MustBeSet() (error, bool) {
	if !f.setInMessage {
		return ErrFieldNotSetInMessage, false
	}
	return nil, true
}

func (f Field) MustBeSetIfInMask() (error, bool) {
	if !f.setInMessage && !f.inMask {
		return nil, true
	}

	if !f.setInMessage && f.inMask {
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

func NewField(path string, value interface{}, inMask, setInMessage bool) *Field {
	return &Field{
		Path:         path,
		Value:        value,
		inMask:       inMask,
		setInMessage: setInMessage,
	}
}

func (f Field) Zero() bool {
	return f.Value == nil || reflect.ValueOf(f.Value).IsZero()
}

// HasTrait implements PolicySubject.
func (f Field) HasTrait(t SubjectTrait) bool {
	switch t.Type() {
	case NotZero:
		return !f.Zero()
	default:
		return true
	}
}

// ID implements PolicySubject.
func (f Field) ID() string {
	return f.Path
}
