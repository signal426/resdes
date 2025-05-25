package soldr

import (
	"fmt"
	"reflect"

	"github.com/google/go-cmp/cmp"
)

var (
	ErrMsgFieldNotComparable string = "expected type %v but got %v"
	ErrMsgFieldsNotEqual     string = "expected value to be equal to %v but received %v"
	ErrMsgFieldsEqual        string = "expected value to not be equal to %v"
	ErrMsgCannotBeZero       string = "expected non-zero but got %v"
)

func isZero(i any) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return v.IsNil() || v.IsZero()
	default:
		return v.IsZero()
	}
}

type Field struct {
	path      string
	value     any
	inMask    bool
	zero      bool
	policy    Policy
	condition Condition
	cmpTo     any
}

func NewField(path string, value any, inMask bool, policy Policy, condition Condition, cmpTo any) *Field {
	return &Field{
		path:      path,
		value:     value,
		inMask:    inMask,
		policy:    policy,
		condition: condition,
		zero:      isZero(value),
		cmpTo:     cmpTo,
	}
}

func (f Field) Validate() error {
	if f.condition == InMask && !f.inMask {
		return nil
	}
	if f.policy == NonZero {
		if f.zero {
			return fmt.Errorf(ErrMsgCannotBeZero, f.value)
		}
		return nil
	}
	eq, err := f.checkEquals()
	if err != nil {
		return err
	}
	if f.policy == NotEqualTo && eq {
		return fmt.Errorf(ErrMsgFieldsEqual, f.cmpTo)
	}
	if f.policy == MustEqual && !eq {
		return fmt.Errorf(ErrMsgFieldsNotEqual, f.cmpTo, f.value)
	}
	return nil
}

func (f Field) GetValue() any {
	return f.value
}

func (f Field) Zero() bool {
	return f.zero
}

func (f Field) ID() string {
	return f.path
}

func (f Field) InMask() bool {
	return f.inMask
}

func (f Field) checkEquals() (bool, error) {
	fieldCmpType := reflect.TypeOf(f.value)
	compareToType := reflect.TypeOf(f.cmpTo)
	if fieldCmpType != compareToType {
		return false, fmt.Errorf(ErrMsgFieldNotComparable, fieldCmpType, compareToType)
	}
	return cmp.Equal(f.value, f.cmpTo), nil
}
