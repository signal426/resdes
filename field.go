package soldr

import (
	"fmt"
	"reflect"
)

var ErrMsgFieldNotComparable string = "expected type %s but got %s"

func isZero(i interface{}) bool {
	return i == nil || reflect.ValueOf(i).IsZero()
}

func (f Field) IsEqualTo(compareTo interface{}) (bool, error) {
	if !f.isComparable(compareTo) {
		return false, fmt.Errorf(ErrMsgFieldNotComparable, reflect.TypeOf(f.Value), reflect.TypeOf(compareTo))
	}
	return true, nil
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

func (f Field) ID() string {
	return f.Path
}

func (f Field) InMask() bool {
	return f.inMask
}
