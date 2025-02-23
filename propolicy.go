package soldr

import (
	"context"
	"errors"

	"google.golang.org/protobuf/proto"
)

var _ TraitPolicy = (*policy)(nil)

type policy struct {
	// the traits the subject must have to be evaluated to passing
	traits SubjectTrait

	// a state-assertion func on the policy subject
	condition ConditionAssertion

	// the policy subject
	subject PolicySubject
}

func NewPolicy(traits SubjectTrait, condition ConditionAssertion, subject PolicySubject) *policy {
	return &policy{
		traits:    traits,
		condition: condition,
		subject:   subject,
	}
}

func (p *policy) EvaluateSubject() *PolicyFault {
	// check the field conditions against the policy conditions
	err, cont := p.condition()
	if err != nil {
		return &PolicyFault{
			SubjectID: p.subject.ID(),
			Error:     err,
		}
	}

	// if no err, but we get a signal not to move on (optional trait assertions), move on
	if !cont {
		return nil
	}

	// if the field has met the policy conditions, check its traits
	if err := p.assertSubjectTraits(p.subject, p.traits); err != nil {
		return &PolicyFault{
			SubjectID: p.subject.ID(),
			Error:     err,
		}
	}

	return nil
}

func (p *policy) assertSubjectTraits(subject PolicySubject, mustHave SubjectTrait) error {
	if mustHave == nil {
		return nil
	}

	// if the targe trait is valid and the subject doesn't have it, check for OR conds
	if mustHave.Valid() && !subject.HasTrait(mustHave) {

		// if we have an or, keep going
		if mustHave.Or().Valid() {
			return p.assertSubjectTraits(subject, mustHave.Or())
		}

		// else, we're done checking
		return errors.New(mustHave.FaultString())
	}
	// if there's an and condition, keep going
	// else, we're done
	if mustHave.And().Valid() {
		return p.assertSubjectTraits(subject, mustHave.And())
	}

	return nil
}

var _ ActionPolicy[proto.Message] = (*actionPolicy[proto.Message])(nil)

type actionPolicy[T proto.Message] struct {
	*policy
	a Action[T]
}

func NewActionPolicy[T proto.Message](condition ConditionAssertion, subject PolicySubject, action Action[T]) *actionPolicy[T] {
	return &actionPolicy[T]{
		policy: NewPolicy(nil, condition, subject),
		a:      action,
	}
}

func (p *actionPolicy[T]) RunAction(ctx context.Context, msg T) *PolicyFault {
	// check the field conditions against the policy conditions
	err, cont := p.condition()
	if err != nil {
		return &PolicyFault{
			SubjectID: p.subject.ID(),
			Error:     err,
		}
	}

	if !cont {
		return nil
	}

	// if the field has met the policy conditions, check its traits
	if err := p.a(ctx, msg); err != nil {
		return &PolicyFault{
			SubjectID: p.subject.ID(),
			Error:     err,
		}
	}

	return nil
}
