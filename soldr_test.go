package soldr

import (
	"context"
	"errors"
	"fmt"
	"testing"

	proplv1 "buf.build/gen/go/signal426/propl/protocolbuffers/go/propl/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

type MyErrResultHandler struct{}

func (my MyErrResultHandler) HandleErrs(errs []Fault) error {
	var errString string
	for _, err := range errs {
		errString += fmt.Sprintf("%s: %s\n", err.Field, err.Err)
	}
	return errors.New(errString)
}

func TestFieldPolicies(t *testing.T) {
	t.Run("it should validate non-zero", func(t *testing.T) {
		// arrange
		req := &proplv1.CreateUserRequest{
			User: &proplv1.User{},
		}

		p := ForSubject(req).
			AssertNonZero("user", req.GetUser()).
			AssertNonZero("user.first_name", req.GetUser().GetFirstName())

		// act
		err := p.E(context.Background())

		// assert
		assert.Error(t, err)
	})

	t.Run("it should evaluate a custom function", func(t *testing.T) {
		// arrange
		req := &proplv1.CreateUserRequest{
			User: &proplv1.User{
				FirstName: "Bob",
			},
		}

		p := ForSubject(req).
			CustomValidation(func(ctx context.Context, msg *proplv1.CreateUserRequest, faults FaultMap) error {
				if msg.GetUser().GetFirstName() == "bob" {
					faults.Add("user.first_name", errors.New("cannot be bob"))
				}
				return nil
			})

		// act
		err := p.E(context.Background())

		// assert
		assert.Error(t, err)
	})

	t.Run("it should evaluate not eq", func(t *testing.T) {
		// arrange
		req := &proplv1.CreateUserRequest{
			User: &proplv1.User{
				FirstName: "Bob",
			},
		}

		p := ForSubject(req).
			AssertNotEqualTo("user.first_name", req.GetUser().GetFirstName(), "bob")

		// act
		err := p.E(context.Background())

		// assert
		assert.Error(t, err)
	})

	t.Run("it should validate a complex structure", func(t *testing.T) {
		// arrange
		req := &proplv1.UpdateUserRequest{
			User: &proplv1.User{
				FirstName: "bob",
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
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNotEqualTo("user.id", req.GetUser().GetId(), "bob").
			AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress()).
			AssertNonZeroWhenInMask("user.primary_address.line1", req.GetUser().GetPrimaryAddress().GetLine1()).
			AssertNotEqualToWhenInMask("user.primary_address.last_name", req.GetUser().GetPrimaryAddress().GetLine2(), "b")

		// act
		err := p.E(context.Background())

		// assert
		assert.Error(t, err)
	})

	t.Run("it should validate custom optional action", func(t *testing.T) {
		// arrange
		req := &proplv1.UpdateUserRequest{
			User: &proplv1.User{
				FirstName: "bob",
				PrimaryAddress: &proplv1.Address{
					Line1: "a",
					Line2: "b",
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"first_name", "last_name", "line1"},
			},
		}

		// ForSubject(request, options...) instantiates the evaluator
		p := ForSubject(req).
			// Specify all of the field paths that should not be equal to their zero value
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNonZero("some.fake", nil).
			AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
			AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
			AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress()).
			CustomValidation(func(ctx context.Context, msg *proplv1.UpdateUserRequest, fieldFaults FaultMap) error {
				if req.GetUser().GetPrimaryAddress().GetLine1() == "a" {
					fieldFaults.Add("user.primary_address_line1", errors.New("cannot be a"))
				}
				return nil
			})

		// act
		// call this before running the evaluation in order to substitute your own error result handler
		// to do things like custom formatting
		err := p.E(context.Background())

		// assert
		assert.Error(t, err)
	})

	t.Run("it should run a custom action before request field validation", func(t *testing.T) {
		// arrange
		authUpdate := func(ctx context.Context) error {
			callerId := ctx.Value("callerId")
			if callerId == nil || callerId == "" {
				return errors.New("caller id is empty")
			}
			return nil
		}

		req := &proplv1.UpdateUserRequest{
			User: &proplv1.User{
				FirstName: "bob",
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
				return authUpdate(ctx)
			}).
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
			AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
			AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress())

		// act
		err := p.E(context.Background())

		// assert
		assert.Error(t, err)
	})

	t.Run("it should run a custom action if validation successful", func(t *testing.T) {
		// arrange
		authorizeUpdate := func(ctx context.Context) error {
			userId := ctx.Value("userId")
			if userId == nil {
				return errors.New("cannot resolve request due to missing user id")
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
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
			CustomValidation(func(ctx context.Context, msg *proplv1.UpdateUserRequest, fieldFaults FaultMap) error {
				if msg.GetUser().GetPrimaryAddress() == nil {
					fieldFaults.Add("user.primar_address", errors.New("should not be nil"))
				}
				return nil
			}).
			// runs immediately after field valiadtion
			AfterValidation(func(ctx context.Context, msg *proplv1.UpdateUserRequest, fieldValidationResult ValidationResult) error {
				// can check the field validation results before using a value if need be or just run this function
				return authorizeUpdate(ctx)
			}).
			// runs last if all was successful
			OnSuccess(func(ctx context.Context, msg *proplv1.UpdateUserRequest) error {
				msg.User.LastName = "NA"
				return nil
			})
		// act
		err := p.E(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, "NA", req.GetUser().GetLastName())
	})
}
