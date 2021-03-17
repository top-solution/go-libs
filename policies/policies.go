package controllers

import (
	"context"

	"github.com/ory/ladon"
	"gitlab.com/top-solution/collins-go-libs/authorizer"
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
