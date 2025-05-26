package resdes

import (
	"reflect"

	"github.com/google/go-cmp/cmp"
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
	path           string
	pathNormalized string
	value          any
	inMask         bool
	zero           bool
	policy         Policy
	condition      Condition
	cmpTo          any
}

func NewField(path string, value any, policy Policy, condition Condition, cmpTo any, paths map[string]struct{}) *Field {
	normalizedPath := NormalizePath(path)
	return &Field{
		path:           path,
		value:          value,
		inMask:         IsPathInMask(normalizedPath, paths),
		policy:         policy,
		condition:      condition,
		zero:           isZero(value),
		pathNormalized: normalizedPath,
		cmpTo:          cmpTo,
	}
}

func (f Field) Validate() error {
	if f.condition == InMask && !f.inMask {
		return nil
	}
	if f.policy == NonZero {
		if f.zero {
			return newFieldMustNotBeZeroFailedErr(f.path, f.value)
		}
		return nil
	}
	eq, err := f.checkEquals()
	if err != nil {
		return err
	}
	if f.policy == NotEqualTo && eq {
		return newFieldMustNotEqualFailedErr(f.path, f.cmpTo)
	}
	if f.policy == MustEqual && !eq {
		return newFieldMustEqualFailedErr(f.path, f.cmpTo, f.value)
	}
	return nil
}

func (f Field) Policy() Policy {
	return f.policy
}

func (f Field) CompareTo() any {
	return f.cmpTo
}

func (f Field) Condition() Condition {
	return f.condition
}

func (f Field) Path(normalized bool) string {
	if normalized {
		return f.pathNormalized
	}
	return f.path
}

func (f Field) Value() any {
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
		return false, newFieldsNotComparableErr(f.path, fieldCmpType, compareToType)
	}
	return cmp.Equal(f.value, f.cmpTo), nil
}
