## Soldr

A way to set policies on proto fields without writing a ton of `if` statements. This was made for readability and reduction of human error -- although
not slow, it achieves a nice API through reflection and recursion, so not the same scale as the `if` statement route.

###  Usage
Example:
```go
authorizeUpdate := func(_ context.Context, userId string) error {
	if userId != "abc123" {
		return errors.New("can only update user abc123")
	}
	return nil
}

doLogic := func(_ context.Context, msg *proplv1.UpdateUserRequest) error {
	if msg.GetUser().GetLastName() == "" {
		msg.GetUser().LastName = "NA"
	}
	return nil
}

req := &proplv1.UpdateUserRequest{
	User: &proplv1.User{
		FirstName: "bob",
		Id:        "123abc",
		PrimaryAddress: &proplv1.Address{
			Line1: "a",
			Line2: "b",
		},
	},
	UpdateMask: &fieldmaskpb.FieldMask{
		Paths: []string{"first_name", "last_name"},
	},
}

p := ForSubject(req, req.GetUpdateMask().Paths...).
	BeforeValidation(func(ctx context.Context, msg *proplv1.UpdateUserRequest) error {
		return authorizeUpdate(ctx, msg.GetUser().GetId())
	}).
	AssertNonZero("user.id", req.GetUser().GetId()).
	AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
	AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
	AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress()).
	OnSuccess(func(ctx context.Context, msg *proplv1.UpdateUserRequest) error {
		return doLogic(ctx, msg)
	})

// act
err := p.E(context.Background())

// assert
assert.NoError(t, err)
assert.Equal(t, req.GetUser().GetLastName(), "NA")
```
Any field on the message not specified in the request policy does not get evaluated.

