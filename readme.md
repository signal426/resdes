## Response Designer

Removes boilerplate from serving protobuf-based requests.

### Install
`go get github.com/signal426/resdes`

### Field Validation
The default message validator utilizes a fluent API to compose a set of policies and conditions under which a field must exist. Any
exceptions to this policy are added to the `ValidatorErrors` object. To handle fields in a custom way, use the `Validator` function through
the `CustomValidation` API. There can only be one custom function per-instance. To add field-level errors, simply add to the error object passed
in and return nil. For any errors that occur outside of the field-level (i.e. io, etc...), return the error.

### Response Arrangement

#### Auth
The Auth stage is where the first function is called. Any errors returned from this stage will return an error immediately.

#### Validation
The Validation stage is a plug-able stage. It can be executed independent of an arrangement, and can also be passed into compose
an arrangement. Any errors returned from this stage will return an error immediately.

#### Serve
The Serve stage is the last function to be executed and only if any previously declared stages have executed successfully. 

### Error Types
Calling
`.Exec(ctx, request)`
on an arragement returns 2 values:
- the response object
- an error object

All errors implement the error interface, so simply calling `.Error()` will give you the error message that you can wrap however
you'd like for downstream handling. 

### Examples

#### Field validation only
```go
mv := resdes.ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
	AssertNonZero("user.id", req.GetUser().GetId()).
	AssertNotEqualToWhenInMask("user.first_name", req.GetUser().GetFirstName(), "bob").
	AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
	AssertNonZeroWhenInMask("user.primary_address.line1", req.GetUser().GetPrimaryAddress().GetLine1()).
	AssertNotEqualToWhenInMask("user.primary_address.line2", req.GetUser().GetPrimaryAddress().GetLine2(), "b").
	CustomValidation(func(ctx context.Context, uur *v1.UpdateUserRequest, ve *ValidationErrors) error {
		if uur.GetUser().GetId() == "abc123" {
			ve.AddFieldErr("user.id", errors.New("user id cannot be abc123"))
		}
		// return nil if only adding field-level errors
		return nil
	})
```

#### Full request handling
```go
resp, err := resdes.Arrange[*v1.UpdateUserRequest, *v1.UpdateUserResponse]().
	WithAuth(func(ctx context.Context, _ *v1.UpdateUserRequest) error {
		return authUpdate(ctx)
	}).
	WithValidate(resdes.ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
		AssertNonZero("user", req.GetUser()).
		AssertNonZero("user.id", req.GetUser().GetId()),
	).
	WithServe(func(ctx context.Context, uur *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
		return someBusinessLogic(ctx, uur)
	}).Exec(context.Background(), req)
```
