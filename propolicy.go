package soldr

var _ TraitPolicy = (*policy)(nil)

type policy struct {
	// the traits the subject must have to be evaluated to passing
	trait SubjectTrait

	// a state-assertion func on the policy subject
	condition ConditionAssertion

	// the policy subject
	subject PolicySubject
}

func NewPolicy(trait SubjectTrait, condition ConditionAssertion, subject PolicySubject) *policy {
	return &policy{
		trait:     trait,
		condition: condition,
		subject:   subject,
	}
}

func (p *policy) EvaluateSubject() *PolicyFault {
	// check the field conditions against the policy conditions
	pass, msg := p.condition()
	if !pass {
		return &PolicyFault{
			SubjectID: p.subject.ID(),
			Message:   msg,
		}
	}

	// if no err, but we get a signal not to move on (optional trait assertions), move on
	if p.trait == Traitless() {
		return nil
	}

	// if the field has met the policy conditions, check its traits
	if msg := p.assertSubjectTrait(p.subject, p.trait); msg != "" {
		return &PolicyFault{
			SubjectID: p.subject.ID(),
			Message:   msg,
		}
	}

	return nil
}

func (p *policy) assertSubjectTrait(subject PolicySubject, mustHave SubjectTrait) string {
	if mustHave == nil {
		return ""
	}

	if !subject.HasTrait(mustHave) {
		return mustHave.FaultString()
	}

	return ""
}
