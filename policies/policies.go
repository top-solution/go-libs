package policies

import (
	"context"

	"github.com/ory/ladon"
	"gitlab.com/top-solution/go-libs/authorizer"
	"gitlab.com/top-solution/go-libs/keys"
	goa "goa.design/goa/v3/pkg"
)

type Policies struct {
	*authorizer.LadonAuthorizer
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
