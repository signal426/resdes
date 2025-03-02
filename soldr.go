package soldr

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
)

const (
	ErrMsgFieldCannotBeZero    = "field value cannot be zero value for type"
	ErrMsgFieldCannotHaveValue = "field cannot have value"
)

type (
	FieldFaultMap              map[string]string
	PreAction[T proto.Message] func(ctx context.Context, msg T, validationResult *ValidationResult)
	Action[T proto.Message]    func(ctx context.Context, msg T, validationResult *ValidationResult)
)

func (f FieldFaultMap) Set(key string, details string) {
	f[key] = details
}

type ValidationResult struct {
	FieldFaults           FieldFaultMap `json:"field_faults"`
	RequestFailureMessage string        `json:"message"`
	RequestFailureDetails string        `json:"details"`
}

func NewValidationResult(fieldFaults FieldFaultMap) *ValidationResult {
	return &ValidationResult{
		FieldFaults: fieldFaults,
	}
}

func (v ValidationResult) Continue() bool {
	return v.RequestFailureMessage == ""
}

func (v *ValidationResult) ContainsFaultForField(path string) bool {
	if v == nil || v.FieldFaults == nil || len(v.FieldFaults) == 0 {
		return false
	}
	_, ok := v.FieldFaults[path]
	return ok
}

func (v *ValidationResult) AddFieldFault(path string, msg string) {
	v.FieldFaults[path] = msg
}

func (v *ValidationResult) SetResponseErr(msg, details string) {
	v.RequestFailureMessage = msg
	v.RequestFailureDetails = details
}

type Subject[T proto.Message] struct {
	// custom actions to run before fields are evaluated
	// any error from a pre-field eval returns early
	initAction PreAction[T]

	// custom actions to run after fields are evaluated
	successAction Action[T]

	// custom action to run regardless if an error occurred
	postAction Action[T]

	// custom validation func
	customValidations []Action[T]

	fieldFaults FieldFaultMap

	// paths is list of fields that are being evaluated if a field mask is supplied
	paths map[string]struct{}

	// the message we are processing
	message T

	// errors from the validation builder to be surfaced on evaluation
	configFaults error
}

// For creates a new policy aggregate for the specified message that can be built upon using the
// builder methods.
func ForSubject[T proto.Message](subject T, fieldMask ...string) *Subject[T] {
	return &Subject[T]{
		paths:             getPathsFromMask(fieldMask...),
		message:           subject,
		customValidations: []Action[T]{},
	}
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

// HasNonZeroField pass in a list of fields that must not be equal to their
// zero value
//
// example: sue := HasNonZeroFields("user.id", "user.first_name")
func (s *Subject[T]) AssertNonZero(path string, value interface{}) *Subject[T] {
	// create a new field policy subject
	field := NewField(path, value, s.isFieldInMask(path))
	if field.Zero() {
		s.fieldFaults.Set(field.ID(), ErrMsgFieldCannotBeZero)
	}

	return s
}

func (s *Subject[T]) addConfigErr(err error) {
	if s.configFaults == nil {
		s.configFaults = err
	} else {
		errors.Join(s.configFaults, err)
	}
}

// HasNonZeroField pass in a list of fields that must not be equal to their
// zero value
//
// example: sue := HasNonZeroFields("user.id", "user.first_name")
func (s *Subject[T]) AssertNotEqualTo(path string, value interface{}, notEqualTo interface{}) *Subject[T] {
	// create a new field policy subject
	field := NewField(path, value, s.isFieldInMask(path))
	eq, err := field.IsEqualTo(notEqualTo)
	if err != nil {
		s.addConfigErr(err)
		return s
	}

	if eq {
		s.fieldFaults.Set(field.ID(), fmt.Sprintf(ErrMsgFieldCannotHaveValue+": %v", notEqualTo))
	}

	return s
}

// HasNonZeroFieldsWhen pass in a list of field conditions if you want to customize the conditions under which
// a field non-zero evaluation is triggered
//
// example: sue := HasNonZeroFieldsWhen(IfInMask("user.first_name"), Always("user.first_name"))
func (s *Subject[T]) AssertNonZeroWhenInMask(path string, value interface{}) *Subject[T] {
	// create a new field policy subject
	field := NewField(path, value, s.isFieldInMask(path))
	if !field.InMask() {
		return s
	}
	if field.Zero() {
		s.fieldFaults.Set(field.ID(), ErrMsgFieldCannotBeZero)
	}

	return s
}

// HasNonZeroFieldsWhen pass in a list of field conditions if you want to customize the conditions under which
// a field non-zero evaluation is triggered
//
// example: sue := HasNonZeroFieldsWhen(IfInMask("user.first_name"), Always("user.first_name"))
func (s *Subject[T]) AssertNotEqualToWhenInMask(path string, value interface{}, notEqualTo interface{}) *Subject[T] {
	// create a new field policy subject
	field := NewField(path, value, s.isFieldInMask(path))
	if !field.InMask() {
		return s
	}
	eq, err := field.IsEqualTo(notEqualTo)
	if err != nil {
		s.addConfigErr(err)
		return s
	}

	if eq {
		s.fieldFaults.Set(field.ID(), fmt.Sprintf(ErrMsgFieldCannotHaveValue+": %v", notEqualTo))
	}

	return s
}

func (s *Subject[T]) CustomValidation(act Action[T]) *Subject[T] {
	s.customValidations = append(s.customValidations, act)
	return s
}

func (s *Subject[T]) BeforeValidation(act PreAction[T]) *Subject[T] {
	s.initAction = act
	return s
}

func (s *Subject[T]) AfterValidation(act Action[T]) *Subject[T] {
	s.postAction = act
	return s
}

func (s *Subject[T]) OnSuccess(act Action[T]) *Subject[T] {
	s.successAction = act
	return s
}

func (s *Subject[T]) isFieldInMask(path string) bool {
	if s.paths == nil {
		return false
	}
	_, inMask := s.paths[path]
	return inMask
}

// E shorthand for Evaluate
func (s *Subject[T]) E(ctx context.Context) (*ValidationResult, error) {
	return s.Evaluate(ctx)
}

// Evaluate checks each declared policy and returns an error describing
// each infraction. If a precheck is specified and returns an error, this exits
// and field policies are not evaluated.
//
// To use your own infractionsHandler, specify a handler using WithInfractionsHandler.
func (s *Subject[T]) Evaluate(ctx context.Context) (*ValidationResult, error) {
	// return an err if there were any invalid configurations applied
	if s.configFaults != nil {
		return nil, s.configFaults
	}

	validationResult := NewValidationResult(s.fieldFaults)
	if s.initAction != nil {
		s.initAction(ctx, s.message, validationResult)
		if !validationResult.Continue() {
			return validationResult, nil
		}
	}

	if len(s.customValidations) > 0 {
		for _, c := range s.customValidations {
			c(ctx, s.message, validationResult)
			if !validationResult.Continue() {
				return validationResult, nil
			}
		}
	}

	if s.postAction != nil {
		s.postAction(ctx, s.message, validationResult)
		if !validationResult.Continue() {
			return validationResult, nil
		}
	}

	if len(s.fieldFaults) == 0 && validationResult.Continue() && s.successAction != nil {
		s.successAction(ctx, s.message, validationResult)
	}

	if len(validationResult.FieldFaults) > 0 && validationResult.RequestFailureMessage == "" {
		validationResult.SetResponseErr("request message contains invalid values", "")
	}

	return validationResult, nil
}
