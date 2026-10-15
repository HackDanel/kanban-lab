package httpserver

import (
	"context"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

// handlers implements the generated api.StrictServerInterface on top of the service.
type handlers struct {
	svc     *service.Service
	cookies cookies
}

var _ api.StrictServerInterface = (*handlers)(nil)

func pageOf(limit, offset *int) service.Page {
	var p service.Page
	if limit != nil {
		p.Limit = *limit
	}
	if offset != nil {
		p.Offset = *offset
	}
	return p
}

func deref[T any](p *T) T {
	var zero T
	if p == nil {
		return zero
	}
	return *p
}

// --- auth ---

func (h *handlers) Register(ctx context.Context, req api.RegisterRequestObject) (api.RegisterResponseObject, error) {
	user, err := h.svc.Register(ctx, string(req.Body.Email), req.Body.Name, req.Body.Password)
	if err != nil {
		return nil, err
	}
	return api.Register201JSONResponse(toUser(user)), nil
}

func (h *handlers) Login(ctx context.Context, req api.LoginRequestObject) (api.LoginResponseObject, error) {
	session, err := h.svc.Login(ctx, string(req.Body.Email), req.Body.Password)
	if err != nil {
		return nil, err
	}
	h.cookies.setSession(exchangeFrom(ctx).w, session)
	return api.Login200JSONResponse(toAuth(session)), nil
}

func (h *handlers) RefreshToken(ctx context.Context, req api.RefreshTokenRequestObject) (api.RefreshTokenResponseObject, error) {
	ex := exchangeFrom(ctx)
	if err := checkCSRF(ex.r, req.Params.XCSRFToken); err != nil {
		return nil, err
	}
	session, err := h.svc.Refresh(ctx, cookieValue(ex.r, refreshCookie))
	if err != nil {
		if domain.KindOf(err) == domain.KindUnauthorized {
			h.cookies.clear(ex.w)
		}
		return nil, err
	}
	h.cookies.setSession(ex.w, session)
	return api.RefreshToken200JSONResponse(toAuth(session)), nil
}

func (h *handlers) Logout(ctx context.Context, req api.LogoutRequestObject) (api.LogoutResponseObject, error) {
	ex := exchangeFrom(ctx)
	if err := checkCSRF(ex.r, req.Params.XCSRFToken); err != nil {
		return nil, err
	}
	if err := h.svc.Logout(ctx, cookieValue(ex.r, refreshCookie)); err != nil {
		return nil, err
	}
	h.cookies.clear(ex.w)
	return api.Logout204Response{}, nil
}

func (h *handlers) GetMe(ctx context.Context, _ api.GetMeRequestObject) (api.GetMeResponseObject, error) {
	user, err := h.svc.Me(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return api.GetMe200JSONResponse(toUser(user)), nil
}

// --- users ---

func (h *handlers) ListUsers(ctx context.Context, req api.ListUsersRequestObject) (api.ListUsersResponseObject, error) {
	users, total, err := h.svc.ListUsers(ctx, actorFrom(ctx), deref(req.Params.Q), pageOf(req.Params.Limit, req.Params.Offset))
	if err != nil {
		return nil, err
	}
	return api.ListUsers200JSONResponse{Items: mapSlice(users, toUser), Total: int(total)}, nil
}

func (h *handlers) SetUserRole(ctx context.Context, req api.SetUserRoleRequestObject) (api.SetUserRoleResponseObject, error) {
	user, err := h.svc.SetRole(ctx, actorFrom(ctx), req.UserId, domain.Role(req.Body.Role))
	if err != nil {
		return nil, err
	}
	return api.SetUserRole200JSONResponse(toUser(user)), nil
}

// --- projects ---

func (h *handlers) ListProjects(ctx context.Context, _ api.ListProjectsRequestObject) (api.ListProjectsResponseObject, error) {
	projects, err := h.svc.ListProjects(ctx, actorFrom(ctx))
	if err != nil {
		return nil, err
	}
	return api.ListProjects200JSONResponse(mapSlice(projects, toProject)), nil
}

func (h *handlers) CreateProject(ctx context.Context, req api.CreateProjectRequestObject) (api.CreateProjectResponseObject, error) {
	project, err := h.svc.CreateProject(ctx, actorFrom(ctx), service.ProjectInput{
		Key: req.Body.Key, Name: req.Body.Name, Description: deref(req.Body.Description),
	})
	if err != nil {
		return nil, err
	}
	return api.CreateProject201JSONResponse(toProject(project)), nil
}

func (h *handlers) GetProject(ctx context.Context, req api.GetProjectRequestObject) (api.GetProjectResponseObject, error) {
	project, err := h.svc.GetProject(ctx, actorFrom(ctx), req.ProjectId)
	if err != nil {
		return nil, err
	}
	return api.GetProject200JSONResponse(toProject(project)), nil
}

func (h *handlers) UpdateProject(ctx context.Context, req api.UpdateProjectRequestObject) (api.UpdateProjectResponseObject, error) {
	project, err := h.svc.UpdateProject(ctx, actorFrom(ctx), req.ProjectId, service.ProjectPatch{
		Name: req.Body.Name, Description: req.Body.Description,
	})
	if err != nil {
		return nil, err
	}
	return api.UpdateProject200JSONResponse(toProject(project)), nil
}

func (h *handlers) DeleteProject(ctx context.Context, req api.DeleteProjectRequestObject) (api.DeleteProjectResponseObject, error) {
	if err := h.svc.DeleteProject(ctx, actorFrom(ctx), req.ProjectId); err != nil {
		return nil, err
	}
	return api.DeleteProject204Response{}, nil
}

func (h *handlers) ListProjectMembers(ctx context.Context, req api.ListProjectMembersRequestObject) (api.ListProjectMembersResponseObject, error) {
	members, err := h.svc.ListMembers(ctx, actorFrom(ctx), req.ProjectId)
	if err != nil {
		return nil, err
	}
	return api.ListProjectMembers200JSONResponse(mapSlice(members, toUser)), nil
}

func (h *handlers) AddProjectMember(ctx context.Context, req api.AddProjectMemberRequestObject) (api.AddProjectMemberResponseObject, error) {
	if err := h.svc.AddMember(ctx, actorFrom(ctx), req.ProjectId, req.Body.UserId); err != nil {
		return nil, err
	}
	return api.AddProjectMember204Response{}, nil
}

func (h *handlers) RemoveProjectMember(ctx context.Context, req api.RemoveProjectMemberRequestObject) (api.RemoveProjectMemberResponseObject, error) {
	if err := h.svc.RemoveMember(ctx, actorFrom(ctx), req.ProjectId, req.UserId); err != nil {
		return nil, err
	}
	return api.RemoveProjectMember204Response{}, nil
}

// --- labels ---

func (h *handlers) ListLabels(ctx context.Context, req api.ListLabelsRequestObject) (api.ListLabelsResponseObject, error) {
	labels, err := h.svc.ListLabels(ctx, actorFrom(ctx), req.ProjectId)
	if err != nil {
		return nil, err
	}
	return api.ListLabels200JSONResponse(mapSlice(labels, toLabel)), nil
}

func (h *handlers) CreateLabel(ctx context.Context, req api.CreateLabelRequestObject) (api.CreateLabelResponseObject, error) {
	label, err := h.svc.CreateLabel(ctx, actorFrom(ctx), req.ProjectId, req.Body.Name, req.Body.Color)
	if err != nil {
		return nil, err
	}
	return api.CreateLabel201JSONResponse(toLabel(label)), nil
}

func (h *handlers) UpdateLabel(ctx context.Context, req api.UpdateLabelRequestObject) (api.UpdateLabelResponseObject, error) {
	label, err := h.svc.UpdateLabel(ctx, actorFrom(ctx), req.LabelId, service.LabelPatch{Name: req.Body.Name, Color: req.Body.Color})
	if err != nil {
		return nil, err
	}
	return api.UpdateLabel200JSONResponse(toLabel(label)), nil
}

func (h *handlers) DeleteLabel(ctx context.Context, req api.DeleteLabelRequestObject) (api.DeleteLabelResponseObject, error) {
	if err := h.svc.DeleteLabel(ctx, actorFrom(ctx), req.LabelId); err != nil {
		return nil, err
	}
	return api.DeleteLabel204Response{}, nil
}
