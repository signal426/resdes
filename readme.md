## Soldr

A way to set policies on proto fields without writing a ton of `if` statements. This was made for readability and reduction of human error -- although
not slow, it achieves a nice API through reflection and recursion, so not the same scale as the `if` statement route.

###  Usage
Example:
```go
// arrange
authorizeUpdate := func(userId string) error {
	if userId != "abc123" {
		return errors.New("can only update user abc123")
	}
	return nil
}

doLogic := func(_ context.Context, _ *proplv1.UpdateUserRequest) error {
	// do some user update logic here
	return nil
}

// create a set of policies for the current request
p := ForSubject(req, req.GetUpdateMask().Paths...).
	// an action that gets run before request validation and returns early if an err occurrs
	WithValidationGateAction(func(ctx context.Context, msg *proplv1.UpdateUserRequest) error {
		return authorizeUpdate(msg.GetUser().GetId())
	}).
	// create a formatted error for the given label and value if value is zero
	AssertNonZero("user.id", req.GetUser().GetId()).
	// can gracefully handle non-existent labels or nil values
	AssertNonZero("some.fake", nil).
	// create a formatted error for the given label and value if label is included in mask and value is zero
	AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
	AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
	AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress()).
	WithPostValidationAction(func(ctx context.Context, msg *proplv1.UpdateUserRequest) error {
		return doLogic(ctx, msg)
	})

// act
err := p.E(context.Background())
```
Any field on the message not specified in the request policy does not get evaluated.

