package soldr

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
)

// Fault map for field label to validation failure details
type FieldFault struct {
	Path string
	Err  error
}

// Some action to run during request processing
type Validator[T proto.Message] func(context.Context, T, *ValidationResult) error

// Handler to run if no validation faults. Returns some response object
type Handler[T proto.Message, U any] func(context.Context, T) (U, error)

// InitAction to run before field validation. An error returned from this acrtio
type Auther[T proto.Message] func(context.Context, T) error

// ValidationResult is the result of the pipeline execution
type ValidationResult struct {
	fieldFaults []*FieldFault
	resultIdx   map[string]int
}

func NewValidationResult() *ValidationResult {
	return &ValidationResult{
		fieldFaults: []*FieldFault{},
		resultIdx:   make(map[string]int),
	}
}

func (v *ValidationResult) ToErr() error {
	if len(v.fieldFaults) == 0 {
		return nil
	}
	var err error
	for _, fault := range v.fieldFaults {
		err = errors.Join(err, fmt.Errorf("%s: %w", fault.Path, fault.Err))
	}
	return err
}

func (v *ValidationResult) Failed() bool {
	return v != nil && len(v.fieldFaults) > 0
}

func (v *ValidationResult) AppendFieldFault(path string, err error) {
	v.addErr(path, err)
}

func (v *ValidationResult) AppendFieldFaultErrStr(path string, details string) {
	v.addErr(path, errors.New(details))
}

func (v *ValidationResult) addErr(path string, err error) {
	idx, ok := v.resultIdx[path]
	if ok {
		v.fieldFaults[idx].Err = errors.Join(v.fieldFaults[idx].Err, err)
	} else {
		v.fieldFaults = append(v.fieldFaults, &FieldFault{Path: path, Err: err})
		v.resultIdx[path] = len(v.fieldFaults) - 1
	}
}

type MessageValidator[T proto.Message] interface {
	Exec(context.Context, T) error
}

var _ MessageValidator[proto.Message] = (*DefaultMessageValidator[proto.Message])(nil)

type DefaultMessageValidator[T proto.Message] struct {
	// custom validation func
	customValidation Validator[T]

	// result of executing this line of actions
	result *ValidationResult

	// paths is list of fields that are being evaluated if a field mask is supplied
	paths map[string]struct{}

	// fields to validate
	fields []*Field
}

func ForMessage[T proto.Message](fieldMask ...string) *DefaultMessageValidator[T] {
	return &DefaultMessageValidator[T]{
		paths:  getPathsFromMask(fieldMask...),
		result: NewValidationResult(),
		fields: []*Field{},
	}
}

func (s *DefaultMessageValidator[T]) AssertNonZero(path string, value any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, s.isFieldInMask(path), NonZero, Always, nil))
	return s
}

func (s *DefaultMessageValidator[T]) AssertNotEqualTo(path string, value any, notEqualTo any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, s.isFieldInMask(path), NotEqualTo, Always, notEqualTo))
	return s
}

func (s *DefaultMessageValidator[T]) AssertEqualTo(path string, value any, equalTo any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, s.isFieldInMask(path), MustEqual, Always, equalTo))
	return s
}

func (s *DefaultMessageValidator[T]) AssertNonZeroWhenInMask(path string, value any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, s.isFieldInMask(path), NonZero, InMask, nil))
	return s
}

func (s *DefaultMessageValidator[T]) AssertNotEqualToWhenInMask(path string, value any, notEqualTo any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, s.isFieldInMask(path), NotEqualTo, InMask, notEqualTo))
	return s
}

func (s *DefaultMessageValidator[T]) AssertEqualToWhenInMask(path string, value any, equalTo any) *DefaultMessageValidator[T] {
	s.fields = append(s.fields, NewField(path, value, s.isFieldInMask(path), MustEqual, InMask, equalTo))
	return s
}

func (s *DefaultMessageValidator[T]) CustomValidation(act Validator[T]) *DefaultMessageValidator[T] {
	s.customValidation = act
	return s
}

func (s *DefaultMessageValidator[T]) Exec(ctx context.Context, message T) error {
	if len(s.fields) > 0 {
		for _, field := range s.fields {
			if err := field.Validate(); err != nil {
				s.result.AppendFieldFault(field.ID(), err)
			}
		}
	}

	if s.customValidation != nil {
		if err := s.customValidation(ctx, message, s.result); err != nil {
			return fmt.Errorf("an error occurred during custom message validation: %w", err)
		}
	}

	if s.result.Failed() {
		return s.result.ToErr()
	}

	return nil
}

func (s *DefaultMessageValidator[T]) isFieldInMask(path string) bool {
	if s.paths == nil {
		return false
	}
	_, inMask := s.paths[path]
	return inMask
}

func getPathsFromMask(fieldMask ...string) map[string]struct{} {
	if fieldMask == nil || len(fieldMask) == 0 {
		return nil
	}
	paths := make(map[string]struct{})
	for _, f := range fieldMask {
		paths[f] = struct{}{}
	}
	return paths
}

// Arrangement represents different actions to take during the
// execution of serving some request
type Arrangement[T proto.Message, U any] struct {
	// action to run before running field validations
	Auth Auther[T]

	// field processor
	Validate MessageValidator[T]

	// logic to run if all validations completed successfully --
	// typically some business logic
	Handle Handler[T, U]
}

func Arrange[T proto.Message, U any]() *Arrangement[T, U] {
	return &Arrangement[T, U]{}
}

func (r *Arrangement[T, U]) WithAuth(act Auther[T]) *Arrangement[T, U] {
	r.Auth = act
	return r
}

func (r *Arrangement[T, U]) WithValidate(fv MessageValidator[T]) *Arrangement[T, U] {
	r.Validate = fv
	return r
}

func (r *Arrangement[T, U]) WithHandle(act Handler[T, U]) *Arrangement[T, U] {
	r.Handle = act
	return r
}

func (s *Arrangement[T, U]) Exec(ctx context.Context, message T) (U, error) {
	// process the init action, if err, return
	var res U
	if s.Auth != nil {
		if err := s.Auth(ctx, message); err != nil {
			return res, err
		}
	}

	// validate fields if we have basic field validations
	if s.Validate != nil {
		if err := s.Validate.Exec(ctx, message); err != nil {
			return res, err
		}
	}

	// if no field faults, run success action
	if s.Handle != nil {
		return s.Handle(ctx, message)
	}

	return res, nil
}
