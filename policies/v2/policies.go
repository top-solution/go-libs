package policies

import (
	"context"

	"github.com/ory/ladon"
	"github.com/top-solution/go-libs/authorizer/v2"
	"github.com/top-solution/go-libs/keys/v2"
)

type Policies struct {
	*authorizer.LadonAuthorizer
}

func (p *Policies) CheckUserRoleAllowed(ctx context.Context, appId string) error {
	var err error
	roles := keys.ClaimsFromContext(ctx).Role
	if keys.ClaimsFromContext(ctx).AppRoleMap != nil {
		roles = keys.ClaimsFromContext(ctx).AppRoleMap[appId]
	}
	for _, r := range roles {
		err = p.IsUserAllowed(ctx, &ladon.Request{
			Context: ladon.Context{
				"role": r,
			},
		})
		if err == nil {
			return nil
		}
	}
	return err
}
