package soldr

import (
	"google.golang.org/protobuf/proto"
)

// FaultMap is the result of a policy
// execution
type FaultMap map[string]string

func (f FaultMap) Add(key string, msg string) {
	_, ok := f[key]
	if ok {
		f[key] = msg
		return
	}
	f[key] = msg
}

func (f FaultMap) Contains(key string) bool {
	_, ok := f[key]
	return ok
}

// ConditionAssertion is a function that can be applied to a policy
// with which it can ensure the subject is in the right state
// before running an evaluation
type ConditionAssertion func() (bool, string)

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
	// some error string describing the validation error
	FaultString() string
	// the trait type
	Type() TraitType
	// get the value that the trait cannot have
	NotEqualTo() interface{}
}

// A TraitPolicy is a set of rules that the specified subjects
// must uphold
type TraitPolicy interface {
	EvaluateSubject() *PolicyFault
}

type PolicyFault struct {
	SubjectID string
	Message   string
}

// policy manager maintains the list of configured policies
// and the context with which it was constructed (if supplied).
//
// it also contains a reference to the field store where it can
// fetch policy subjects for evaluation and set them on creation.
type policyManager[T proto.Message] struct {
	policies []TraitPolicy
}

func NewPolicyManager[T proto.Message]() *policyManager[T] {
	return &policyManager[T]{
		policies: make([]TraitPolicy, 0, 3),
	}
}

func (p *policyManager[T]) AddTraitPolicy(traitPolicy TraitPolicy) {
	p.policies = append(p.policies, traitPolicy)
}

func (p *policyManager[T]) Apply() FaultMap {
	fm := make(FaultMap)
	for _, policy := range p.policies {
		if policy == nil {
			continue
		}
		if pf := policy.EvaluateSubject(); pf != nil {
			fm[pf.SubjectID] = pf.Message
		}
	}
	return fm
}
