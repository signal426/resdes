package resdes

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// Auther a function to run before validation or request serve
type Auther[T proto.Message] func(context.Context, T) error

// Validator function accepts a context, some message, and pointer to current list of ValidationErrors
type Validator[T proto.Message] func(context.Context, T, *ValidationErrors) error

// Server a function to run if all other stages of a request arrangement have passed
type Server[T proto.Message, U any] func(context.Context, T) (U, error)

// MessageValidator responsible for validating supplied fields and returning
// the results in a ValidationErrors object
type MessageValidator[T proto.Message] interface {
	Exec(context.Context, T) *ValidationErrors
}

var _ MessageValidator[proto.Message] = (*DefaultMessageValidator[proto.Message])(nil)

type DefaultMessageValidator[T proto.Message] struct {
	// custom validation func. Only one can be set per validator instance
	customValidation Validator[T]

	// paths is list of fields that are being evaluated if a field mask is supplied
	paths map[string]struct{}

	// fields to validate
	fields []*Field
}

// ForMessage creates a new DefaultMessageValidator
// Accepts paths from a field mask if available
func ForMessage[T proto.Message](fieldMask ...string) *DefaultMessageValidator[T] {
	return &DefaultMessageValidator[T]{
		paths:  GetPathsFromMask(fieldMask...),
		fields: []*Field{},
	}
}

// AssertNonZero assert that the value for the supplied field path is not a zero-value
func (s *DefaultMessageValidator[T]) AssertNonZero(path string, value any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, NonZero, Always, nil, s.paths))
	return s
}

// AssertNotEqualTo assert that the value for the supplied field path is not equal to the supplied target value
func (s *DefaultMessageValidator[T]) AssertNotEqualTo(path string, value any, notEqualTo any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, NotEqualTo, Always, notEqualTo, s.paths))
	return s
}

// AssertEqualTo assert that the value for the supplied fialed path is equal to the supplied target value
func (s *DefaultMessageValidator[T]) AssertEqualTo(path string, value any, equalTo any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, MustEqual, Always, equalTo, s.paths))
	return s
}

// AssertNonZeroWhenInMask same as AssertNonZero, but only executes if the supplied path is in the field mask
func (s *DefaultMessageValidator[T]) AssertNonZeroWhenInMask(path string, value any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, NonZero, InMask, nil, s.paths))
	return s
}

// AssertNotEqualToWhenInMask same as AssertNotEqualTo, but only executes if the supplied path is in the field mask
func (s *DefaultMessageValidator[T]) AssertNotEqualToWhenInMask(path string, value any, notEqualTo any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, NotEqualTo, InMask, notEqualTo, s.paths))
	return s
}

// AssertEqualToWhenInMask same as AssertEqualTo, but only executes if the supplied path is in the field mask
func (s *DefaultMessageValidator[T]) AssertEqualToWhenInMask(path string, value any, equalTo any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, MustEqual, InMask, equalTo, s.paths))
	return s
}

// CustomValidation is a custom validation function. There can only be one per-validator instance.
// To add field-level errors to the existing list of field validation errors (in the case regular Assertxxx functions are used),
// add the errors to the ValidationErrors object and return nil.
//
// In the case that a non-field level error occurs, return the err
func (s *DefaultMessageValidator[T]) CustomValidation(act Validator[T]) *DefaultMessageValidator[T] {
	s.customValidation = act
	return s
}

// Exec executes in the following order:
// 1. Custom validation function if it exists
// 2. Field-level assertion functions
func (s *DefaultMessageValidator[T]) Exec(ctx context.Context, message T) *ValidationErrors {
	errs := NewValidationErrors()
	if s.customValidation != nil {
		// if the validation error is simply returned, continue
		if err := s.customValidation(ctx, message, errs); err != nil && !errors.Is(err, errs) {
			errs.SetCustomValidationErr(fmt.Errorf("an error occurred during custom message validation: %w", err))
		}
	}

	if len(s.fields) > 0 {
		for _, field := range s.fields {
			if err := field.Validate(); err != nil {
				errs.addFieldErr(field, err)
			}
		}
	}

	if errs.HasErrors() {
		return errs
	}

	return nil
}

// Arrangement represents different actions to take during the
// execution of serving some request
type Arrangement[T proto.Message, U any] struct {
	// action to run before running field validations
	Auth Auther[T]

	// validator to validate the incoming message
	Validate MessageValidator[T]

	// logic to run if all validations completed successfully --
	// typically some business logic
	Serve Server[T, U]
}

// Instantiate a new Arrangement to build
func Arrange[T proto.Message, U any]() *Arrangement[T, U] {
	return &Arrangement[T, U]{}
}

// Add an Auth behavior
func (r *Arrangement[T, U]) WithAuth(act Auther[T]) *Arrangement[T, U] {
	r.Auth = act
	return r
}

// Add a Validate behavior
func (r *Arrangement[T, U]) WithValidate(fv MessageValidator[T]) *Arrangement[T, U] {
	r.Validate = fv
	return r
}

// Add a Serve behavior
func (r *Arrangement[T, U]) WithServe(act Server[T, U]) *Arrangement[T, U] {
	r.Serve = act
	return r
}

// Exec runs in the following order:
// 1. Auth
// 2. Validate
// 3. Serve
// The function exits if any error is encountered at any stage
func (s *Arrangement[T, U]) Exec(ctx context.Context, message T) (U, *Error) {
	// process the init action, if err, return
	var res U
	serr := &Error{}
	if s.Auth != nil {
		if err := s.Auth(ctx, message); err != nil {
			serr.SetAuthError(err)
			return res, serr
		}
	}

	// validate fields if we have basic field validations
	if s.Validate != nil {
		if err := s.Validate.Exec(ctx, message); err != nil {
			serr.SetValidationErrors(err)
			return res, serr
		}
	}

	// if no field faults, run success action
	if s.Serve != nil {
		var err error
		res, err = s.Serve(ctx, message)
		if err != nil {
			serr.SetServeError(err)
			return res, serr
		}
	}

	return res, nil
}
