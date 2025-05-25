package soldr

import (
	"context"
	"errors"
	"fmt"
	"testing"

	v1 "github.com/signal426/soldr/test_protos/gen/test_protos/soldr/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestFieldValidations(t *testing.T) {
	t.Run("it should validate non-zero", func(t *testing.T) {
		// arrange
		req := &v1.CreateUserRequest{}

		expected := NewValidationResult()
		expected.AppendFieldFaultErrStr("user", fmt.Sprintf(ErrMsgCannotBeZero, "<nil>"))
		expected.AppendFieldFaultErrStr("user.first_name", fmt.Sprintf(ErrMsgCannotBeZero, ""))

		mv := ForMessage[*v1.CreateUserRequest]().
			AssertNonZero("user", req.GetUser()).
			AssertNonZero("user.first_name", req.GetUser().GetFirstName())

		// act
		err := mv.Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expected.ToErr().Error(), err.Error())
	})

	t.Run("it should evaluate a custom function", func(t *testing.T) {
		// arrange
		req := &v1.CreateUserRequest{
			User: &v1.User{
				FirstName: "bob",
			},
		}

		expected := NewValidationResult()
		expected.AppendFieldFaultErrStr("user.first_name", "cannot be bob")

		mv := ForMessage[*v1.CreateUserRequest]().
			CustomValidation(func(ctx context.Context, msg *v1.CreateUserRequest, validationResult *ValidationResult) error {
				if msg.GetUser().GetFirstName() == "bob" {
					validationResult.AppendFieldFaultErrStr("user.first_name", "cannot be bob")
				}
				return nil
			})

			// act
		err := mv.Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expected.ToErr(), err)
	})

	t.Run("it should evaluate not eq", func(t *testing.T) {
		// arrange
		req := &v1.CreateUserRequest{
			User: &v1.User{
				FirstName: "Bob",
			},
		}

		mv := ForMessage[*v1.CreateUserRequest]().
			AssertNotEqualTo("user.first_name", req.GetUser().GetFirstName(), "bob")

		// act
		err := mv.Exec(context.Background(), req)

		// assert
		assert.NoError(t, err)
	})

	t.Run("it should validate a complex structure", func(t *testing.T) {
		// arrange
		req := &v1.UpdateUserRequest{
			User: &v1.User{
				FirstName: "bob",
				PrimaryAddress: &v1.Address{
					Line1: "a",
					Line2: "b",
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"first_name", "last_name"},
			},
		}

		expected := NewValidationResult()
		expected.AppendFieldFaultErrStr("user.id", fmt.Sprintf(ErrMsgCannotBeZero, ""))
		expected.AppendFieldFaultErrStr("user.first_name", fmt.Sprintf(ErrMsgFieldsEqual, "bob"))

		mv := ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNotEqualToWhenInMask("user.first_name", req.GetUser().GetId(), "bob").
			AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress()).
			AssertNonZeroWhenInMask("user.primary_address.line1", req.GetUser().GetPrimaryAddress().GetLine1()).
			AssertNotEqualToWhenInMask("user.primary_address.line2", req.GetUser().GetPrimaryAddress().GetLine2(), "b")

		// act
		err := mv.Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expected.ToErr().Error(), err.Error())
	})

	t.Run("it should validate custom optional action", func(t *testing.T) {
		// arrange
		req := &v1.UpdateUserRequest{
			User: &v1.User{
				FirstName: "bob",
				PrimaryAddress: &v1.Address{
					Line1: "a",
					Line2: "b",
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"first_name", "last_name", "line1"},
			},
		}

		expected := NewValidationResult()
		expected.AppendFieldFaultErrStr("user.primary_address_line1", "cannot be a")
		expected.AppendFieldFaultErrStr("some.fake", fmt.Sprintf(ErrMsgCannotBeZero, "<nil>"))
		expected.AppendFieldFaultErrStr("user.last_name", fmt.Sprintf(ErrMsgCannotBeZero, ""))

		mv := ForMessage[*v1.UpdateUserRequest]().
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNonZero("some.fake", nil).
			AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
			AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
			AssertNonZeroWhenInMask("user.primary_address", req.GetUser().GetPrimaryAddress()).
			CustomValidation(func(ctx context.Context, msg *v1.UpdateUserRequest, validationResult *ValidationResult) error {
				if req.GetUser().GetPrimaryAddress().GetLine1() == "a" {
					validationResult.AppendFieldFaultErrStr("user.primary_address_line1", "cannot be a")
				}
				return nil
			})

		// act
		err := mv.Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expected.ToErr().Error(), err.Error())
	})
}

func TestArrangements(t *testing.T) {
	t.Run("it should run a custom action before request field validation", func(t *testing.T) {
		// arrange
		expectedErr := errors.New("caller id is empty")
		authUpdate := func(ctx context.Context) error {
			callerId := ctx.Value("callerId")
			if callerId == nil || callerId == "" {
				return expectedErr
			}
			return nil
		}

		req := &v1.UpdateUserRequest{
			User: &v1.User{
				FirstName: "bob",
				PrimaryAddress: &v1.Address{
					Line1: "a",
					Line2: "b",
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"first_name", "last_name"},
			},
		}

		// act
		resp, err := Arrange[*v1.UpdateUserRequest, *v1.UpdateUserResponse]().
			WithAuth(func(ctx context.Context, _ *v1.UpdateUserRequest) error {
				return authUpdate(ctx)
			}).
			WithValidate(ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
				AssertNonZero("user", req.GetUser()).
				AssertNonZero("user.id", req.GetUser().GetId()),
			).
			WithHandle(func(ctx context.Context, uur *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
				return nil, nil
			}).Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expectedErr.Error(), err.Error())
		assert.Nil(t, resp)
	})

	t.Run("it should run a custom action if validation successful", func(t *testing.T) {
		// arrange
		authorizeUpdate := func(_ context.Context) error {
			return nil
		}

		uppercaseName := func(_ context.Context, msg *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
			user := proto.Clone(msg.GetUser()).(*v1.User)
			user.FirstName = "Bob"
			return &v1.UpdateUserResponse{
				User: user,
			}, nil
		}

		req := &v1.UpdateUserRequest{
			User: &v1.User{
				FirstName: "bob",
				Id:        "abc123",
				PrimaryAddress: &v1.Address{
					Line1: "a",
					Line2: "b",
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"first_name", "last_name"},
			},
		}

		rp := Arrangement[*v1.UpdateUserRequest, *v1.UpdateUserResponse]{
			Auth: func(ctx context.Context, uur *v1.UpdateUserRequest) error {
				return authorizeUpdate(ctx)
			},
			Validate: ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
				AssertNonZero("user.id", req.GetUser().GetId()).
				AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
				CustomValidation(func(ctx context.Context, msg *v1.UpdateUserRequest, validationResult *ValidationResult) error {
					if msg.GetUser().GetPrimaryAddress() == nil {
						validationResult.AppendFieldFaultErrStr("user.primary_address", "should not be nil")
					}
					return nil
				}),
			Handle: func(ctx context.Context, cur *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
				return uppercaseName(ctx, cur)
			},
		}

		// act
		res, err := rp.Exec(context.Background(), req)

		// assert
		assert.NoError(t, err)
		assert.NotNil(t, res)
		assert.Equal(t, "Bob", res.GetUser().GetFirstName())
	})
}
