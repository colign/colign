package comment

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	commentv1 "github.com/gobenpark/colign/gen/proto/comment/v1"
	"github.com/gobenpark/colign/gen/proto/comment/v1/commentv1connect"
	"github.com/gobenpark/colign/internal/auth"
	"github.com/gobenpark/colign/internal/models"
)

type ConnectHandler struct {
	service           *Service
	jwtManager        *auth.JWTManager
	apiTokenValidator auth.APITokenValidator
}

var _ commentv1connect.CommentServiceHandler = (*ConnectHandler)(nil)

func NewConnectHandler(service *Service, jwtManager *auth.JWTManager, apiTokenValidator auth.APITokenValidator) *ConnectHandler {
	return &ConnectHandler{service: service, jwtManager: jwtManager, apiTokenValidator: apiTokenValidator}
}

func (h *ConnectHandler) extractClaims(ctx context.Context, header string) (*auth.Claims, error) {
	claims, err := auth.ResolveFromHeader(h.jwtManager, h.apiTokenValidator, ctx, header)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}
	return claims, nil
}

func (h *ConnectHandler) CreateComment(ctx context.Context, req *connect.Request[commentv1.CreateCommentRequest]) (*connect.Response[commentv1.CreateCommentResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	comment, err := h.service.Create(
		ctx,
		req.Msg.ChangeId,
		req.Msg.DocumentType,
		req.Msg.QuotedText,
		req.Msg.Body,
		claims.UserID,
		req.Msg.MentionedUserIds,
	)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&commentv1.CreateCommentResponse{
		Comment: commentToProto(comment),
	}), nil
}

func (h *ConnectHandler) ListComments(ctx context.Context, req *connect.Request[commentv1.ListCommentsRequest]) (*connect.Response[commentv1.ListCommentsResponse], error) {
	comments, err := h.service.List(ctx, req.Msg.ChangeId, req.Msg.DocumentType)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoComments := make([]*commentv1.Comment, len(comments))
	for i, c := range comments {
		protoComments[i] = commentToProto(&c)
	}

	return connect.NewResponse(&commentv1.ListCommentsResponse{
		Comments: protoComments,
	}), nil
}

func (h *ConnectHandler) ResolveComment(ctx context.Context, req *connect.Request[commentv1.ResolveCommentRequest]) (*connect.Response[commentv1.ResolveCommentResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	comment, err := h.service.Resolve(ctx, req.Msg.CommentId, claims.UserID, claims.OrgID)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&commentv1.ResolveCommentResponse{
		Comment: commentToProto(comment),
	}), nil
}

func (h *ConnectHandler) DeleteComment(ctx context.Context, req *connect.Request[commentv1.DeleteCommentRequest]) (*connect.Response[commentv1.DeleteCommentResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	if err := h.service.Delete(ctx, req.Msg.CommentId, claims.UserID, claims.OrgID); err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if errors.Is(err, ErrNotAuthorized) {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&commentv1.DeleteCommentResponse{}), nil
}

func (h *ConnectHandler) CreateReply(ctx context.Context, req *connect.Request[commentv1.CreateReplyRequest]) (*connect.Response[commentv1.CreateReplyResponse], error) {
	claims, err := h.extractClaims(ctx, req.Header().Get("Authorization"))
	if err != nil {
		return nil, err
	}

	reply, err := h.service.CreateReply(ctx, req.Msg.CommentId, req.Msg.Body, claims.UserID, claims.OrgID, req.Msg.MentionedUserIds)
	if err != nil {
		if errors.Is(err, ErrCommentNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&commentv1.CreateReplyResponse{
		Reply: replyToProto(reply),
	}), nil
}

func commentToProto(c *models.Comment) *commentv1.Comment {
	proto := &commentv1.Comment{
		Id:           c.ID,
		ChangeId:     c.ChangeID,
		DocumentType: c.DocumentType,
		QuotedText:   c.QuotedText,
		Body:         c.Body,
		UserId:       c.UserID,
		Resolved:     c.Resolved,
		CreatedAt:    timestamppb.New(c.CreatedAt),
		UpdatedAt:    timestamppb.New(c.UpdatedAt),
	}
	if c.ResolvedBy != nil {
		proto.ResolvedBy = *c.ResolvedBy
	}
	if c.User != nil {
		proto.UserName = c.User.Name
	}

	proto.Replies = make([]*commentv1.Reply, len(c.Replies))
	for i, r := range c.Replies {
		proto.Replies[i] = replyToProto(&r)
	}

	return proto
}

func replyToProto(r *models.CommentReply) *commentv1.Reply {
	proto := &commentv1.Reply{
		Id:        r.ID,
		CommentId: r.CommentID,
		Body:      r.Body,
		UserId:    r.UserID,
		CreatedAt: timestamppb.New(r.CreatedAt),
	}
	if r.User != nil {
		proto.UserName = r.User.Name
	}
	return proto
}
