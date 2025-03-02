package soldr

import (
	"context"

	"google.golang.org/protobuf/proto"
)

const (
	prevalErr  = "the message failed prevalidation"
	postValErr = "the message failed postvalidation"
)

// output format of the errors
type Format uint32

const (
	Default Format = iota
	JSON
)

type ValidationResult struct {
	FieldFaults           FaultMap
	RequestFailureMessage string
	RequestFailureDetails string
}

func NewValidationResult(fieldFaults FaultMap) *ValidationResult {
	return &ValidationResult{
		FieldFaults: fieldFaults,
	}
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

type PreAction[T proto.Message] func(ctx context.Context, msg T) (error, bool)

type Action[T proto.Message] func(ctx context.Context, msg T, validationResult ValidationResult) error

type Subject[T proto.Message] struct {
	// custom actions to run before fields are evaluated
	// any error from a pre-field eval returns early
	initAction PreAction[T]

	// custom actions to run after fields are evaluated
	successAction Action[T]

	// custom action to run regardless if an error occurred
	postAction Action[T]

	// policy manager for executing policies
	pm *policyManager[T]

	// custom validation func
	customValidations []Action[T]

	// the handler for the faults
	fh FaultHandler

	// paths is list of fields that are being evaluated if a field mask is supplied
	paths map[string]struct{}

	// the message we are processing
	message T

	// errors from the validation builder to be surfaced on evaluation
	configFaults []*Fault
}

// For creates a new policy aggregate for the specified message that can be built upon using the
// builder methods.
func ForSubject[T proto.Message](subject T, fieldMask ...string) *Subject[T] {
	return &Subject[T]{
		paths:             getPathsFromMask(fieldMask...),
		pm:                NewPolicyManager[T](),
		message:           subject,
		customValidations: []Action[T]{},
	}
}

func (s *Subject[T]) addConfigFault(err error) {
	if s.configFaults == nil {
		s.configFaults = []*Fault{}
	}
	s.configFaults = append(s.configFaults, ConfigFault(err))
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
func (s *Subject[T]) AssertNonZero(sath string, value interface{}) *Subject[T] {
	// create a new field policy subject
	field := NewField(sath, value, s.isFieldInMask(sath))

	// create the trait policy for the field
	traitPolicy := NewPolicy(NotZeroTrait(), field.MustBeSet, field)

	// add the policy to our manager
	s.pm.AddTraitPolicy(traitPolicy)
	return s
}

// HasNonZeroField pass in a list of fields that must not be equal to their
// zero value
//
// example: sue := HasNonZeroFields("user.id", "user.first_name")
func (s *Subject[T]) AssertNotEqualTo(sath string, value interface{}, notEqualTo interface{}) *Subject[T] {
	// create a new field policy subject
	field := NewField(sath, value, s.isFieldInMask(sath))

	var (
		trait SubjectTrait
		err   error
	)

	// if the comparison value is zero, just use a non-zero trait policy
	if isZero(notEqualTo) {
		trait = NotZeroTrait()
	} else {
		// create the trait
		if !field.isComparable(notEqualTo) {
			s.addConfigFault(ErrComparisonTypeMismatch)
			return s
		}

		trait, err = NotEqualTrait(notEqualTo)
		if err != nil {
			s.addConfigFault(err)
			return s
		}
	}

	// create the policy with the trait
	traitPolicy := NewPolicy(trait, field.MustBeSet, field)

	// add the policy to our manager
	s.pm.AddTraitPolicy(traitPolicy)
	return s
}

// HasNonZeroFieldsWhen pass in a list of field conditions if you want to customize the conditions under which
// a field non-zero evaluation is triggered
//
// example: sue := HasNonZeroFieldsWhen(IfInMask("user.first_name"), Always("user.first_name"))
func (s *Subject[T]) AssertNonZeroWhenInMask(path string, value interface{}) *Subject[T] {
	// create a new field policy subject
	field := NewField(path, value, s.isFieldInMask(path))

	// create the trait policy for the field
	traitPolicy := NewPolicy(NotZeroTrait(), field.MustBeSetIfInMask, field)

	// add the policy to our manager
	s.pm.AddTraitPolicy(traitPolicy)
	return s
}

// HasNonZeroFieldsWhen pass in a list of field conditions if you want to customize the conditions under which
// a field non-zero evaluation is triggered
//
// example: sue := HasNonZeroFieldsWhen(IfInMask("user.first_name"), Always("user.first_name"))
func (s *Subject[T]) AssertNotEqualToWhenInMask(path string, value interface{}, notEqualTo interface{}) *Subject[T] {
	// create a new field policy subject
	field := NewField(path, value, s.isFieldInMask(path))

	var (
		trait SubjectTrait
		err   error
	)

	// if the comparison value is zero, just use a non-zero trait policy
	if isZero(notEqualTo) {
		trait = NotZeroTrait()
	} else {
		// create the trait
		trait, err = NotEqualTrait(ErrComparisonTypeMismatch)
		if err != nil {
			s.addConfigFault(err)
			return s
		}
	}

	// create the trait policy for the field
	traitPolicy := NewPolicy(trait, field.MustBeSetIfInMask, field)

	// add the policy to our manager
	s.pm.AddTraitPolicy(traitPolicy)
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

// CustomErrResultHandler call this before calling E() or Evaluate() if you want to override
// the errors that are output from the policy execution
func (s *Subject[T]) CustomFaultHandler(e FaultHandler) *Subject[T] {
	s.fh = e
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
func (s *Subject[T]) E(ctx context.Context) error {
	return s.Evaluate(ctx)
}

func (s *Subject[T]) beforeValidation(ctx context.Context) (*Fault, bool) {
	// evaluate the global pre-checks
	if s.initAction != nil {
		if err, cont := s.initAction(ctx, s.message); err != nil {
			return RequestFault(err), cont
		}
	}

	return nil, true
}

func (s *Subject[T]) afterValidation(ctx context.Context, validationResult ValidationResult) *Fault {
	// evaluate the global pre-checks
	if s.postAction != nil {
		if err := s.postAction(ctx, s.message, validationResult); err != nil {
			return RequestFault(err)
		}
	}

	return nil
}

func (s *Subject[T]) onSuccess(ctx context.Context, validationResult ValidationResult) *Fault {
	// evaluate the global pre-checks
	if s.successAction != nil {
		if err := s.successAction(ctx, s.message, validationResult); err != nil {
			return RequestFault(err)
		}
	}

	return nil
}

func (s *Subject[T]) runCustomValidations(ctx context.Context, validationResult ValidationResult) []*Fault {
	faults := []*Fault{}
	if len(s.customValidations) > 0 {
		for _, c := range s.customValidations {
			if err := c(ctx, s.message, validationResult); err != nil {
				faults = append(faults, CustomFault(err))
			}
		}
	}
	return faults
}

func (s *Subject[T]) err(f []*Fault) error {
	if s.fh == nil {
		s.fh = newDefaultFaultHandler()
	}
	if len(f) == 0 {
		return nil
	}
	return s.fh.ToError(f)
}

// Evaluate checks each declared policy and returns an error describing
// each infraction. If a precheck is specified and returns an error, this exits
// and field policies are not evaluated.
//
// To use your own infractionsHandler, specify a handler using WithInfractionsHandler.
func (s *Subject[T]) Evaluate(ctx context.Context) error {
	// return an err if there were any invalid configurations applied
	if len(s.configFaults) > 0 {
		return s.err(s.configFaults)
	}

	// init action if supplied
	fault, cont := s.beforeValidation(ctx)
	if fault != nil && !cont {
		return s.err([]*Fault{fault})
	}

	// assert field traits based on their condition in the message
	faults := []*Fault{}
	policyFaults := s.pm.Apply()
	if customFaults := s.runCustomValidations(ctx, policyFaults); len(customFaults) > 0 {
		faults = append(faults, customFaults...)
	}

	// get all the field faults
	for subject, fault := range policyFaults {
		faults = append(faults, FieldFault(subject, fault))
	}

	// if there's a post-validation action, run that
	if err := s.afterValidation(ctx, policyFaults); err != nil {
		faults = append(faults, err)
	}

	// if no faults, run the success action
	if len(faults) == 0 {
		if successFault := s.onSuccess(ctx); successFault != nil {
			faults = append(faults, successFault)
		}
	}

	return s.err(faults)
}
