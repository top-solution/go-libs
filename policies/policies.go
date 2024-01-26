package policies

import (
	"context"

	"github.com/ory/ladon"
	"github.com/top-solution/go-libs/authorizer"
	"github.com/top-solution/go-libs/keys"
	goa "goa.design/goa/v3/pkg"
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

func (p *Policies) IsUserAllowed(ctx context.Context, req *ladon.Request) error {
	if req.Action == "" {
		req.Action = ContextAction(ctx)
	}
	if req.Resource == "" {
		req.Resource = ContextService(ctx)
	}
	if req.Subject == "" {
		req.Subject = keys.SubjectFromContext(ctx)
	}

	return p.LadonAuthorizer.IsUserAllowed(ctx, req)
}

func ContextService(ctx context.Context) string {
	if elem, ok := ctx.Value(goa.ServiceKey).(string); ok {
		return elem
	}

	return ""
}

func ContextAction(ctx context.Context) string {
	if elem, ok := ctx.Value(goa.MethodKey).(string); ok {
		return elem
	}

	return ""
}
