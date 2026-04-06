package wiki

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	wikiv1 "github.com/gobenpark/colign/gen/proto/wiki/v1"
	"github.com/gobenpark/colign/gen/proto/wiki/v1/wikiv1connect"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/models"
)

type ConnectHandler struct {
	service           *Service
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
}

var _ wikiv1connect.WikiServiceHandler = (*ConnectHandler)(nil)

func NewConnectHandler(service *Service, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator) *ConnectHandler {
	return &ConnectHandler{service: service, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator}
}

func (h *ConnectHandler) resolveAuth(ctx context.Context, header string) (*auth.Claims, error) {
	return auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, header)
}

func (h *ConnectHandler) CreateWikiPage(ctx context.Context, req *connect.Request[wikiv1.CreateWikiPageRequest]) (*connect.Response[wikiv1.CreateWikiPageResponse], error) {
	claims, err := h.resolveAuth(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	var parentID *uuid.UUID
	if req.Msg.ParentId != "" {
		id, err := uuid.Parse(req.Msg.ParentId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid parent_id: %w", err))
		}
		parentID = &id
	}

	title := req.Msg.Title
	if title == "" {
		title = "Untitled"
	}

	page, err := h.service.CreatePage(ctx, req.Msg.ProjectId, parentID, title, claims.UserID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&wikiv1.CreateWikiPageResponse{
		Page: pageToProto(page),
	}), nil
}

func (h *ConnectHandler) GetWikiPage(ctx context.Context, req *connect.Request[wikiv1.GetWikiPageRequest]) (*connect.Response[wikiv1.GetWikiPageResponse], error) {
	_, err := h.resolveAuth(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	pageID, err := uuid.Parse(req.Msg.PageId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid page_id: %w", err))
	}

	page, err := h.service.GetPage(ctx, req.Msg.ProjectId, pageID)
	if err != nil {
		if errors.Is(err, ErrPageNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&wikiv1.GetWikiPageResponse{
		Page: pageToProto(page),
	}), nil
}

func (h *ConnectHandler) ListWikiPages(ctx context.Context, req *connect.Request[wikiv1.ListWikiPagesRequest]) (*connect.Response[wikiv1.ListWikiPagesResponse], error) {
	_, err := h.resolveAuth(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	pages, err := h.service.ListPages(ctx, req.Msg.ProjectId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoPages := make([]*wikiv1.WikiPage, 0, len(pages))
	for _, p := range pages {
		protoPages = append(protoPages, pageToProto(p))
	}

	return connect.NewResponse(&wikiv1.ListWikiPagesResponse{
		Pages: protoPages,
	}), nil
}

func (h *ConnectHandler) UpdateWikiPage(ctx context.Context, req *connect.Request[wikiv1.UpdateWikiPageRequest]) (*connect.Response[wikiv1.UpdateWikiPageResponse], error) {
	_, err := h.resolveAuth(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	pageID, err := uuid.Parse(req.Msg.PageId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid page_id: %w", err))
	}

	var title, icon, contentJSON, contentText *string
	if req.Msg.Title != "" {
		title = &req.Msg.Title
	}
	if req.Msg.Icon != "" {
		icon = &req.Msg.Icon
	}
	if req.Msg.ContentJson != "" {
		contentJSON = &req.Msg.ContentJson
	}
	if req.Msg.ContentText != "" {
		contentText = &req.Msg.ContentText
	}

	page, err := h.service.UpdatePage(ctx, req.Msg.ProjectId, pageID, title, icon, contentJSON, contentText)
	if err != nil {
		if errors.Is(err, ErrPageNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&wikiv1.UpdateWikiPageResponse{
		Page: pageToProto(page),
	}), nil
}

func (h *ConnectHandler) DeleteWikiPage(ctx context.Context, req *connect.Request[wikiv1.DeleteWikiPageRequest]) (*connect.Response[wikiv1.DeleteWikiPageResponse], error) {
	_, err := h.resolveAuth(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	pageID, err := uuid.Parse(req.Msg.PageId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid page_id: %w", err))
	}

	if err := h.service.DeletePage(ctx, req.Msg.ProjectId, pageID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&wikiv1.DeleteWikiPageResponse{}), nil
}

func (h *ConnectHandler) RestoreWikiPage(ctx context.Context, req *connect.Request[wikiv1.RestoreWikiPageRequest]) (*connect.Response[wikiv1.RestoreWikiPageResponse], error) {
	_, err := h.resolveAuth(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	pageID, err := uuid.Parse(req.Msg.PageId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid page_id: %w", err))
	}

	page, err := h.service.RestorePage(ctx, req.Msg.ProjectId, pageID)
	if err != nil {
		if errors.Is(err, ErrPageNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&wikiv1.RestoreWikiPageResponse{
		Page: pageToProto(page),
	}), nil
}

func (h *ConnectHandler) ReorderWikiPages(ctx context.Context, req *connect.Request[wikiv1.ReorderWikiPagesRequest]) (*connect.Response[wikiv1.ReorderWikiPagesResponse], error) {
	_, err := h.resolveAuth(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	pageID, err := uuid.Parse(req.Msg.PageId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid page_id: %w", err))
	}

	var parentID *uuid.UUID
	if req.Msg.ParentId != "" {
		id, err := uuid.Parse(req.Msg.ParentId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid parent_id: %w", err))
		}
		parentID = &id
	}

	if err := h.service.ReorderPage(ctx, req.Msg.ProjectId, pageID, parentID, int(req.Msg.SortOrder)); err != nil {
		if errors.Is(err, ErrPageNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&wikiv1.ReorderWikiPagesResponse{}), nil
}

func (h *ConnectHandler) UploadWikiImage(ctx context.Context, req *connect.Request[wikiv1.UploadWikiImageRequest]) (*connect.Response[wikiv1.UploadWikiImageResponse], error) {
	claims, err := h.resolveAuth(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	pageID, err := uuid.Parse(req.Msg.PageId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid page_id: %w", err))
	}

	img, err := h.service.UploadImage(ctx, req.Msg.ProjectId, pageID, req.Msg.Filename, req.Msg.ContentType, req.Msg.Data, claims.UserID)
	if err != nil {
		if errors.Is(err, ErrImageTooLarge) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&wikiv1.UploadWikiImageResponse{
		Id:  img.ID,
		Url: fmt.Sprintf("/api/wiki/images/%d", img.ID),
	}), nil
}

func (h *ConnectHandler) GetWikiImage(ctx context.Context, req *connect.Request[wikiv1.GetWikiImageRequest]) (*connect.Response[wikiv1.GetWikiImageResponse], error) {
	_, err := h.resolveAuth(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	img, err := h.service.GetImage(ctx, req.Msg.Id)
	if err != nil {
		if errors.Is(err, ErrImageNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&wikiv1.GetWikiImageResponse{
		Data:        img.Data,
		ContentType: img.ContentType,
		Filename:    img.Filename,
	}), nil
}

func pageToProto(p *models.WikiPage) *wikiv1.WikiPage {
	pp := &wikiv1.WikiPage{
		Id:          p.ID.String(),
		ProjectId:   p.ProjectID,
		Title:       p.Title,
		Icon:        p.Icon,
		SortOrder:   int32(p.SortOrder),
		ContentJson: p.ContentJSON,
		ContentText: p.ContentText,
		CreatedBy:   p.CreatedBy,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
	if p.ParentID != nil {
		pp.ParentId = p.ParentID.String()
	}
	if p.Creator != nil {
		pp.CreatorName = p.Creator.Name
	}
	return pp
}
