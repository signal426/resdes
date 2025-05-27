package resdes

import (
	"context"
	"errors"
	"testing"

	v1 "github.com/signal426/resdes/test_protos/gen/test_protos/resdes/v1"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestFieldValidations(t *testing.T) {
	t.Run("it should assert non-zero", func(t *testing.T) {
		// arrange
		req := &v1.CreateUserRequest{}
		expected := &ValidationErrors{
			FieldErrors: []*FieldError{
				{
					Path:   "user",
					Policy: NonZero,
					Err:    newFieldMustNotBeZeroFailedErr("user", nil),
				},
				{
					Path:   "user.first_name",
					Policy: NonZero,
					Err:    newFieldMustNotBeZeroFailedErr("user.first_name", ""),
				},
			},
		}

		// act
		err := ForMessage[*v1.CreateUserRequest]().
			AssertNonZero("user", req.GetUser()).
			AssertNonZero("user.first_name", req.GetUser().GetFirstName()).
			Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expected.Error(), err.Error())
	})

	t.Run("it should assert not equal", func(t *testing.T) {
		// arrange
		req := &v1.CreateUserRequest{
			User: &v1.User{
				FirstName: "bob",
			},
		}
		expected := &ValidationErrors{
			FieldErrors: []*FieldError{
				{
					Path:   "user.first_name",
					Policy: NotEqualTo,
					Err:    newFieldMustNotEqualFailedErr("user.first_name", req.GetUser().GetFirstName()),
				},
			},
		}

		// act
		err := ForMessage[*v1.CreateUserRequest]().
			AssertNotEqualTo("user.first_name", req.GetUser().GetFirstName(), "bob").
			Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expected.Error(), err.Error())
	})

	t.Run("it should assert equality", func(t *testing.T) {
		// arrange
		req := &v1.CreateUserRequest{
			User: &v1.User{
				FirstName: "Bob",
			},
		}

		expected := &ValidationErrors{
			FieldErrors: []*FieldError{
				{
					Path:   "user.first_name",
					Policy: MustEqual,
					Err:    newFieldMustEqualFailedErr("user.first_name", "bob", req.GetUser().GetFirstName()),
				},
			},
		}

		// act
		err := ForMessage[*v1.CreateUserRequest]().
			AssertEqualTo("user.first_name", req.GetUser().GetFirstName(), "bob").
			Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expected.Error(), err.Error())
	})

	t.Run("it should capture custom validation errors", func(t *testing.T) {
		// arrange
		req := &v1.UpdateUserRequest{
			User: &v1.User{
				FirstName: "Bob",
				Id:        "abc123",
				LastName:  "Bobson",
				PrimaryAddress: &v1.Address{
					Line1: "a",
					Line2: "c",
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"user.firstName", "user.lastName", "user.primaryAddress.line1", "user.primaryAddress.line2"},
			},
		}

		expected := &ValidationErrors{
			FieldErrors: []*FieldError{
				{
					Path:   "user.id",
					Policy: Custom,
					Err:    errors.New("user id cannot be abc123"),
				},
			},
		}

		// act
		err := ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNotEqualToWhenInMask("user.first_name", req.GetUser().GetFirstName(), "bob").
			AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
			AssertNonZeroWhenInMask("user.primary_address.line1", req.GetUser().GetPrimaryAddress().GetLine1()).
			AssertNotEqualToWhenInMask("user.primary_address.line2", req.GetUser().GetPrimaryAddress().GetLine2(), "b").
			CustomValidation(func(ctx context.Context, uur *v1.UpdateUserRequest, ve *ValidationErrors) error {
				if uur.GetUser().GetId() == "abc123" {
					ve.AddFieldErr("user.id", errors.New("user id cannot be abc123"))
				}
				return nil
			}).Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expected.Error(), err.Error())
	})

	t.Run("it should assert on field mask paths", func(t *testing.T) {
		// arrange
		req := &v1.UpdateUserRequest{
			User: &v1.User{
				Id:        "abc123",
				FirstName: "bob",
				PrimaryAddress: &v1.Address{
					Line1: "a",
					Line2: "bca",
				},
			},
			UpdateMask: &fieldmaskpb.FieldMask{
				Paths: []string{"user.firstName", "user.lastName", "user.primaryAddress.line1", "user.primaryAddress.line2"},
			},
		}

		expected := &ValidationErrors{
			FieldErrors: []*FieldError{
				{
					Path:   "user.last_name",
					Policy: NonZero,
					Err:    newFieldMustNotBeZeroFailedErr("user.last_name", ""),
				},
				{
					Path:   "user.primary_address.line1",
					Policy: MustEqual,
					Err:    newFieldMustEqualFailedErr("user.primary_address.line1", "abc", req.GetUser().GetPrimaryAddress().GetLine1()),
				},
				{
					Path:   "user.primary_address.line2",
					Policy: NotEqualTo,
					Err:    newFieldMustNotEqualFailedErr("user.primary_address.line2", req.GetUser().GetPrimaryAddress().GetLine2()),
				},
			},
		}

		// act
		err := ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
			AssertNonZero("user.id", req.GetUser().GetId()).
			AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
			AssertNonZeroWhenInMask("user.last_name", req.GetUser().GetLastName()).
			AssertEqualToWhenInMask("user.primary_address.line1", req.GetUser().GetPrimaryAddress().GetLine1(), "abc").
			AssertNotEqualToWhenInMask("user.primary_address.line2", req.GetUser().GetPrimaryAddress().GetLine2(), "bca").
			Exec(context.Background(), req)

		// assert
		assert.Error(t, err)
		assert.Equal(t, expected.Error(), err.Error())
	})
}

func TestArrangements(t *testing.T) {
	t.Run("it should run auth function", func(t *testing.T) {
		// arrange
		autherr := errors.New("caller id cannot be empty")
		expectedErr := &Error{
			AuthError: NewAuthError(autherr),
		}
		authUpdate := func(ctx context.Context) error {
			callerId := ctx.Value("callerId")
			if callerId == nil || callerId == "" {
				return autherr
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
				Paths: []string{"user.firstName", "user.lastName"},
			},
		}

		// act
		response := Arrange[*v1.UpdateUserRequest, *v1.UpdateUserResponse]().
			WithAuth(func(ctx context.Context, _ *v1.UpdateUserRequest) error {
				return authUpdate(ctx)
			}).
			WithValidate(ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
				AssertNonZero("user", req.GetUser()).
				AssertNonZero("user.id", req.GetUser().GetId()),
			).
			WithServe(func(ctx context.Context, uur *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
				return nil, nil
			}).Exec(context.Background(), req)

		// assert
		assert.Error(t, response.Error())
		assert.Equal(t, expectedErr.Error(), response.Error().Error())
		assert.Nil(t, response.Data())
		assert.Nil(t, response.Error().GetValidationErrors())
		assert.Nil(t, response.Error().GetServeError())
	})

	t.Run("it should run success action if no errors", func(t *testing.T) {
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
				Paths: []string{"user.firstName", "user.lastName"},
			},
		}

		rp := Arrangement[*v1.UpdateUserRequest, *v1.UpdateUserResponse]{
			Auth: func(ctx context.Context, uur *v1.UpdateUserRequest) error {
				return authorizeUpdate(ctx)
			},
			Validate: ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
				AssertNonZero("user.id", req.GetUser().GetId()).
				AssertNonZeroWhenInMask("user.first_name", req.GetUser().GetFirstName()).
				CustomValidation(func(ctx context.Context, msg *v1.UpdateUserRequest, errs *ValidationErrors) error {
					if msg.GetUser().GetPrimaryAddress() == nil {
						errs.AddFieldErr("user.primary_address", errors.New("primary addr cannot be empty"))
					}
					return nil
				}),
			Serve: func(ctx context.Context, cur *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
				return uppercaseName(ctx, cur)
			},
		}

		// act
		response := rp.Exec(context.Background(), req)

		// assert
		assert.Nil(t, response.Error())
		assert.NotNil(t, response.Data())
		assert.Equal(t, "Bob", response.Data().GetUser().GetFirstName())
	})

	t.Run("it should return validation errors in main error object", func(t *testing.T) {
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
				Paths: []string{"user.firstName", "user.lastName"},
			},
		}

		// act
		response := Arrange[*v1.UpdateUserRequest, *v1.UpdateUserResponse]().
			WithAuth(func(ctx context.Context, _ *v1.UpdateUserRequest) error {
				return nil
			}).
			WithValidate(ForMessage[*v1.UpdateUserRequest](req.GetUpdateMask().GetPaths()...).
				AssertNonZero("user", req.GetUser()).
				AssertNonZero("user.id", req.GetUser().GetId()),
			).
			WithServe(func(ctx context.Context, uur *v1.UpdateUserRequest) (*v1.UpdateUserResponse, error) {
				return nil, nil
			}).Exec(context.Background(), req)

		// assert
		assert.Error(t, response.Error())
		assert.Nil(t, response.Data())

		var ae *ValidationErrors
		assert.ErrorAs(t, response.Error(), &ae)
		assert.Len(t, ae.FieldErrors, 1)
		inMap := ae.AsMap()["user.id"]
		assert.NotNil(t, inMap)
	})
}
