package user_v1

import (
	"context"

	converter "github.com/Arkosh744/auth-service-api/internal/converter/user"
	desc "github.com/Arkosh744/auth-service-api/pkg/user_v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (i *Implementation) List(ctx context.Context, _ *emptypb.Empty) (*desc.ListResponse, error) {
	users, err := i.userService.List(ctx)
	if err != nil {
		return nil, err
	}

	return converter.ToUserListDesc(users), nil
}
