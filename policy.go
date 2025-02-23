package soldr

import (
	"context"
	"errors"

	"google.golang.org/protobuf/proto"
)

// Some function triggered by the result of an evaluation whether it be
// a policy or a global evaluation
type Action[T proto.Message] func(ctx context.Context, msg T) error

type PolicyActionOption func(*TraitPolicy)

// FaultMap is the result of a policy
// execution
type FaultMap map[string]error

// ConditionAssertion is a function that can be applied to a policy
// with which it can ensure the subject is in the right state
// before running an evaluation
type ConditionAssertion func() (error, bool)

// a policy subject is a subject that gets evaluated to see:
// 1. what action is configured to occurr if a certain condition is met
// 2. what traits it has if the conditional action results in a trait eval
type PolicySubject interface {
	// some identifier for the subject
	ID() string
	// check for whether or not a policy subject holds a trait
	HasTrait(t SubjectTrait) bool
}

// a trait is an attribute of a policy subject that must
// be true if the policy is a trait evaluation
type SubjectTrait interface {
	// another trait that must exist with this trait
	And() SubjectTrait
	// another trait that must exist if this trait does not
	Or() SubjectTrait
	// some error string describing the validation error
	FaultString() string
	// the trait type
	Type() TraitType
	// state check to report the validity of trait
	Valid() bool
}

// A TraitPolicy is a set of rules that the specified subjects
// must uphold
type TraitPolicy interface {
	EvaluateSubject() *PolicyFault
}

type PolicyFault struct {
	SubjectID string
	Error     error
}

// An action policy is a custom policy that injects the entire message for
// custom evaluation
type ActionPolicy[T proto.Message] interface {
	RunAction(ctx context.Context, msg T) *PolicyFault
}

// policy manager maintains the list of configured policies
// and the context with which it was constructed (if supplied).
//
// it also contains a reference to the field store where it can
// fetch policy subjects for evaluation and set them on creation.
type policyManager[T proto.Message] struct {
	policies       []TraitPolicy
	actionPolicies []ActionPolicy[T]
}

func NewPolicyManager[T proto.Message]() *policyManager[T] {
	return &policyManager[T]{
		policies:       make([]TraitPolicy, 0, 3),
		actionPolicies: make([]ActionPolicy[T], 3),
	}
}

func (p *policyManager[T]) AddTraitPolicy(traitPolicy TraitPolicy) {
	p.policies = append(p.policies, traitPolicy)
}

func (p *policyManager[T]) AddActionPolicy(actionPolicy ActionPolicy[T]) {
	p.actionPolicies = append(p.actionPolicies, actionPolicy)
}

func (p *policyManager[T]) ExecuteTraitPolicies() FaultMap {
	fm := make(FaultMap)
	for _, policy := range p.policies {
		if policy == nil {
			continue
		}
		if pf := policy.EvaluateSubject(); pf != nil {
			fm[pf.SubjectID] = pf.Error
		}
	}
	return fm
}

func (p *policyManager[T]) ExecuteActionPolicies(ctx context.Context, msg T) FaultMap {
	fm := make(FaultMap)
	for _, actionPolicy := range p.actionPolicies {
		if actionPolicy == nil {
			continue
		}
		if pf := actionPolicy.RunAction(ctx, msg); pf != nil {
			fm[pf.SubjectID] = pf.Error
		}
	}
	return fm
}

func (p *policyManager[T]) ExecuteAllPolicies(ctx context.Context, msg T) FaultMap {
	fm := make(FaultMap)
	for _, policy := range p.policies {
		if policy == nil {
			continue
		}
		if pf := policy.EvaluateSubject(); pf != nil {
			fm[pf.SubjectID] = pf.Error
		}
	}
	for _, actionPolicy := range p.actionPolicies {
		if actionPolicy == nil {
			continue
		}
		if pf := actionPolicy.RunAction(ctx, msg); pf != nil {
			if existing, ok := fm[pf.SubjectID]; ok {
				existing = errors.Join(existing, pf.Error)
			} else {
				fm[pf.SubjectID] = pf.Error
			}
		}
	}
	return fm
}
