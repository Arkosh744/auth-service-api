package user_v1

import (
	"context"
	"fmt"

	converter "github.com/Arkosh744/auth-grpc/internal/converter/user"
	desc "github.com/Arkosh744/auth-grpc/pkg/user_v1"
	"github.com/Arkosh744/auth-grpc/pkg/validator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (i *Implementation) Create(ctx context.Context, req *desc.CreateRequest) (*desc.CreateResponse, error) {
	err := validateCreateRequest(req)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "Request validation failed: %v", err)
	}

	user, err := converter.ToUser(req)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Error converting request to user: %v", err)
	}

	err = i.userService.Create(ctx, user)
	if err != nil {
		switch status.Code(err) {
		case codes.Unknown:
			return nil, status.Errorf(codes.Internal, "Error creating user: %v", err)
		default:
			return nil, err
		}
	}

	return &desc.CreateResponse{}, nil
}

func validateCreateRequest(req *desc.CreateRequest) error {
	if !validator.IsPasswordValid(req.GetPassword()) {
		return fmt.Errorf(ErrNotValidPassword)
	}

	if !validator.IsPasswordConfirmed(req.GetPassword(), req.GetPasswordConfirm()) {
		return fmt.Errorf(ErrPasswordConfirmation)
	}

	if !validator.IsValidEmail(req.GetEmail()) {
		return fmt.Errorf(ErrNotValidEmail)
	}

	if !validator.IsUsernameValid(req.GetUsername()) {
		return fmt.Errorf(ErrNotValidUsername)
	}

	return nil
}