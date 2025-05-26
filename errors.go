package resdes

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"google.golang.org/grpc/status"
)

var (
	// ErrFieldComparisonFailedNotComparable returned when an equality policy is applied (e.g. AssertNotEqualTo) to an incompatible type
	ErrFieldComparisonFailedNotComparable = errors.New("equality check failed, types not comparable")
	// ErrFieldMustEqualFailed returned when the supplied value does not match the target value
	ErrFieldMustEqualFailed = errors.New("expected values to be equal")
	// ErrFieldMustEqualFailed returned when the supplied value matches a forbidden value
	ErrFieldMustNotEqualFailed = errors.New("field set to forbidden value")
	// ErrFieldMustNotBeZeroFailed returned when the supplied value matches its type's zero-value
	ErrFieldMustNotBeZeroFailed = errors.New("field set to zero value")
)

func newFieldsNotComparableErr(id string, exp reflect.Type, act reflect.Type) error {
	return fmt.Errorf("field: %s, value: %v, compareTo: %v: %w", id, exp, act, ErrFieldComparisonFailedNotComparable)
}

func newFieldMustEqualFailedErr(id string, exp any, act any) error {
	return fmt.Errorf("field: %s, value: %v, compareTo: %v: %w", id, exp, act, ErrFieldMustEqualFailed)
}

func newFieldMustNotEqualFailedErr(id string, act any) error {
	return fmt.Errorf("field: %s, value: %v: %w", id, act, ErrFieldMustNotEqualFailed)
}

func newFieldMustNotBeZeroFailedErr(id string, act any) error {
	return fmt.Errorf("field: %s, value: %v: %w", id, act, ErrFieldMustNotBeZeroFailed)
}

// AuthError wraps when an error occurs in the auth stage
type AuthError struct {
	Err error
}

func NewAuthError(err error) *AuthError {
	return &AuthError{
		Err: err,
	}
}

func (a *AuthError) Error() string {
	return a.Err.Error()
}

func (a *AuthError) Unwrap() error {
	return a.Err
}

// ServeErr wraps when an error occurs during the handle stage
type ServeErr struct {
	Err error
}

func NewServeError(err error) *ServeErr {
	return &ServeErr{
		Err: err,
	}
}

func (h *ServeErr) Error() string {
	return h.Err.Error()
}

func (h *ServeErr) Unwrap() error {
	return h.Err
}

// Error holds errors for each stage of a request
type Error struct {
	AuthError      *AuthError
	ValidationErrs *ValidationErrors
	ServeError     *ServeErr
}

func (e *Error) Unwrap() error {
	switch {
	case e.GetAuthError() != nil:
		return e.GetAuthError()
	case e.GetValidationErrors() != nil:
		return e.GetValidationErrors()
	case e.GetServeError() != nil:
		return e.GetServeError()
	}
	return nil
}

func (e *Error) SetAuthError(err error) {
	e.AuthError = NewAuthError(err)
}

func (e *Error) SetValidationErrors(errs *ValidationErrors) {
	e.ValidationErrs = errs
}

func (e *Error) SetServeError(err error) {
	e.ServeError = NewServeError(err)
}

func (e *Error) GetAuthError() *AuthError {
	if e == nil {
		return nil
	}
	return e.AuthError
}

func (e *Error) GetValidationErrors() *ValidationErrors {
	if e == nil {
		return nil
	}
	return e.ValidationErrs
}

func (e *Error) GetServeError() *ServeErr {
	if e == nil {
		return nil
	}
	return e.ServeError
}

func (e *Error) ToGrpcStatus() *status.Status {
	return nil
}

func (e *Error) Error() string {
	switch {
	case e.GetAuthError() != nil:
		return e.GetAuthError().Error()
	case e.GetValidationErrors() != nil:
		return e.GetValidationErrors().Error()
	case e.GetServeError() != nil:
		return e.GetServeError().Error()
	}
	return ""
}

type AddFieldValidationErrOption func(*FieldError)

func WithValue(value any) AddFieldValidationErrOption {
	return func(fe *FieldError) {
		fe.Value = value
	}
}

func WithExpectedValue(value any) AddFieldValidationErrOption {
	return func(fe *FieldError) {
		fe.Expected = value
	}
}

// ValidationErrors holds faults for each field evaluated
type ValidationErrors struct {
	FieldErrors           []*FieldError
	CustomValidationError error
	idx                   map[string]int
}

func NewValidationErrors() *ValidationErrors {
	return &ValidationErrors{
		FieldErrors: []*FieldError{},
		idx:         make(map[string]int),
	}
}

func (v *ValidationErrors) addFieldErr(field *Field, err error) {
	v.addErr(FieldErrorFromField(field, err))
}

// AddFieldErr adds an error for a field from a custom eval function
func (v *ValidationErrors) AddFieldErr(path string, err error, options ...AddFieldValidationErrOption) {
	fe := &FieldError{
		Path:   path,
		Err:    err,
		Policy: Custom,
	}
	if len(options) > 0 {
		for _, o := range options {
			o(fe)
		}
	}
	v.addErr(fe)
}

func (v ValidationErrors) Paths() []string {
	paths := make([]string, 0, len(v.FieldErrors))
	for _, f := range v.FieldErrors {
		paths = append(paths, f.Path)
	}
	return paths
}

func (v ValidationErrors) HasErrors() bool {
	return len(v.FieldErrors) > 0 || v.CustomValidationError != nil
}

func (v *ValidationErrors) addErr(fieldErr *FieldError) {
	if v.idx == nil {
		v.idx = make(map[string]int)
	}
	idx, ok := v.idx[fieldErr.Path]
	if ok {
		v.FieldErrors[idx].Err = errors.Join(v.FieldErrors[idx].Err, fieldErr.Err)
	} else {
		v.FieldErrors = append(v.FieldErrors, fieldErr)
		v.idx[fieldErr.Path] = len(v.FieldErrors) - 1
	}
}

func (v *ValidationErrors) SetCustomValidationErr(err error) {
	v.CustomValidationError = err
}

func (v *ValidationErrors) Error() string {
	if v == nil {
		return ""
	}
	var out strings.Builder
	if v.CustomValidationError != nil {
		out.WriteString(v.CustomValidationError.Error() + "\n")
	}
	for _, e := range v.FieldErrors {
		out.WriteString(e.Error() + "\n")
	}
	return out.String()
}

func (v *ValidationErrors) AsMap() map[string]*FieldError {
	if v == nil {
		return nil
	}
	m := make(map[string]*FieldError)
	for _, e := range v.FieldErrors {
		m[e.Path] = e
	}
	return m
}

func (v *ValidationErrors) Unwrap() error {
	if len(v.FieldErrors) == 0 {
		return nil
	}
	errs := make([]error, 0, len(v.FieldErrors))
	for _, e := range v.FieldErrors {
		errs = append(errs, e.Err)
	}
	return errors.Join(errs...)
}

type FieldError struct {
	Path     string
	Policy   Policy
	Value    any
	Expected any
	Err      error
}

func FieldErrorFromField(f *Field, err error) *FieldError {
	return &FieldError{
		Path:     f.ID(),
		Policy:   f.Policy(),
		Value:    f.Value(),
		Expected: f.CompareTo(),
		Err:      err,
	}
}

func (f *FieldError) Error() string {
	if f.Err != nil {
		return fmt.Sprintf("%s failed %s policy: %s", f.Path, f.Policy.String(), f.Err.Error())
	}
	return ""
}

func (f *FieldError) Unwrap() error {
	return f.Err
}
