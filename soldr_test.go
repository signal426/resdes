package soldr

import (
	"context"
	"errors"
	"testing"

	proplv1 "buf.build/gen/go/signal426/propl/protocolbuffers/go/propl/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestFieldPolicies(t *testing.T) {
	t.Run("it should validate non-zero", func(t *testing.T) {
		// arrange
		req := &proplv1.CreateUserRequest{
			User: &proplv1.User{},
		}

		expected := &ValidationResult{
			FieldFaults: map[string]string{
				"user.first_name": ErrMsgFieldCannotBeZero,
			},
		}

		p := ForRequest(req).
			AssertNonZero("user", req.GetUser()).
			AssertNonZero("user.first_name", req.GetUser().GetFirstName())

		// act
		res, err := p.E(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("it should evaluate a custom function", func(t *testing.T) {
		// arrange
		req := &proplv1.CreateUserRequest{
			User: &proplv1.User{
				FirstName: "Bob",
			},
		}

		expected := &ValidationResult{
			FieldFaults: map[string]string{
				"user.first_name": "cannot be bob",
			},
		}

		p := ForRequest(req).
			CustomValidation(func(ctx context.Context, msg *proplv1.CreateUserRequest, validationResult *ValidationResult) {
				if msg.GetUser().GetFirstName() == "bob" {
					validationResult.AddFieldFault("user.first_name", "cannot be bob")
				}
			})

		// act
		res, err := p.E(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
	})

	t.Run("it should evaluate not eq", func(t *testing.T) {
		// arrange
		req := &proplv1.CreateUserRequest{
			User: &proplv1.User{
				FirstName: "Bob",
			},
		}

		expected := &ValidationResult{
			FieldFaults: map[string]string{
				"user.first_name": ErrMsgFieldCannotHaveValue + ": bob",
			},
		}

		p := ForRequest(req).
			AssertNotEqualTo("user.first_name", req.GetUser().GetFirstName(), "bob")

		// act
		res, err := p.E(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
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

		expected := &ValidationResult{
			FieldFaults: map[string]string{
				"": "",
			},
		}

		p := ForRequest(req, req.GetUpdateMask().Paths...).
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNotEqualTo("user.id", req.GetUser().GetId(), "bob").
			AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress()).
			AssertNonZeroWhenInMask("user.primary_address.line1", req.GetUser().GetPrimaryAddress().GetLine1()).
			AssertNotEqualToWhenInMask("user.primary_address.last_name", req.GetUser().GetPrimaryAddress().GetLine2(), "b")

		// act
		res, err := p.E(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
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

		expected := &ValidationResult{
			FieldFaults: map[string]string{
				"some.fake":                  ErrMsgFieldCannotBeZero,
				"user.id":                    ErrMsgFieldCannotBeZero,
				"user.last_name":             ErrMsgFieldCannotBeZero,
				"user.primary_address_line1": ErrMsgFieldCannotHaveValue + ": a",
			},
		}

		// ForSubject(request, options...) instantiates the evaluator
		p := ForRequest(req).
			// Specify all of the field paths that should not be equal to their zero value
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNonZero("some.fake", nil).
			AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
			AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
			AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress()).
			CustomValidation(func(ctx context.Context, msg *proplv1.UpdateUserRequest, validationResult *ValidationResult) {
				if req.GetUser().GetPrimaryAddress().GetLine1() == "a" {
					validationResult.AddFieldFault("user.primary_address_line1", "cannot be a")
				}
			})

		// act
		// call this before running the evaluation in order to substitute your own error result handler
		// to do things like custom formatting
		res, err := p.E(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, expected, res)
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

		p := ForRequest(req, req.GetUpdateMask().Paths...).
			BeforeValidation(func(ctx context.Context, msg *proplv1.UpdateUserRequest, validationResult *ValidationResult) {
				if err := authUpdate(ctx); err != nil {
					validationResult.SetResponseErr("request not authorized", err.Error())
				}
			}).
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
			AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
			AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress())

		// act
		_, err := p.E(context.Background())

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

		p := ForRequest(req, req.GetUpdateMask().Paths...).
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
			CustomValidation(func(ctx context.Context, msg *proplv1.UpdateUserRequest, validationResult *ValidationResult) {
				if msg.GetUser().GetPrimaryAddress() == nil {
					validationResult.AddFieldFault("user.primary_address", "should not be nil")
				}
			}).
			// runs immediately after field valiadtion
			AfterValidation(func(ctx context.Context, msg *proplv1.UpdateUserRequest, validationResult *ValidationResult) {
				// can check the field validation results before using a value if need be or just run this function
				if err := authorizeUpdate(ctx); err != nil {
					validationResult.SetResponseErr("request not authorized", err.Error())
				}
			}).
			// runs last if all was successful
			OnSuccess(func(ctx context.Context, msg *proplv1.UpdateUserRequest, validationResult *ValidationResult) {
				msg.User.LastName = "NA"
			})

		// act
		_, err := p.E(context.Background())

		// assert
		assert.NoError(t, err)
		assert.Equal(t, "NA", req.GetUser().GetLastName())
	})
}
