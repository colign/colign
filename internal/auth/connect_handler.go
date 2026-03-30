package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/http"

	"connectrpc.com/connect"

	authv1 "github.com/gobenpark/colign/gen/proto/auth/v1"
	"github.com/gobenpark/colign/gen/proto/auth/v1/authv1connect"
)

type ConnectHandler struct {
	service      *Service
	oauthService *OAuthService
	cookieOpts   BrowserSessionOptions
}

var _ authv1connect.AuthServiceHandler = (*ConnectHandler)(nil)

func NewConnectHandler(service *Service, oauthService *OAuthService, cookieOpts BrowserSessionOptions) *ConnectHandler {
	return &ConnectHandler{service: service, oauthService: oauthService, cookieOpts: cookieOpts}
}

func (h *ConnectHandler) Register(ctx context.Context, req *connect.Request[authv1.RegisterRequest]) (*connect.Response[authv1.RegisterResponse], error) {
	tokenPair, err := h.service.Register(ctx, RegisterRequest{
		Email:            req.Msg.Email,
		Password:         req.Msg.Password,
		Name:             req.Msg.Name,
		OrganizationName: req.Msg.OrganizationName,
	})
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			return nil, connect.NewError(connect.CodeAlreadyExists, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := connect.NewResponse(&authv1.RegisterResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	})
	AppendBrowserSessionCookies(res.Header(), tokenPair, h.cookieOpts)
	return res, nil
}

func (h *ConnectHandler) Login(ctx context.Context, req *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
	tokenPair, err := h.service.Login(ctx, LoginRequest{
		Email:    req.Msg.Email,
		Password: req.Msg.Password,
	})
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := connect.NewResponse(&authv1.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	})
	AppendBrowserSessionCookies(res.Header(), tokenPair, h.cookieOpts)
	return res, nil
}

func (h *ConnectHandler) RefreshToken(ctx context.Context, req *connect.Request[authv1.RefreshTokenRequest]) (*connect.Response[authv1.RefreshTokenResponse], error) {
	refreshToken := req.Msg.GetRefreshToken()
	if refreshToken == "" {
		cookieReq := &http.Request{Header: http.Header{"Cookie": req.Header().Values("Cookie")}}
		_, refreshToken = BrowserSessionFromRequest(cookieReq)
	}

	tokenPair, err := h.service.RefreshToken(ctx, refreshToken)
	if err != nil {
		if errors.Is(err, ErrInvalidRefreshToken) {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := connect.NewResponse(&authv1.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	})
	AppendBrowserSessionCookies(res.Header(), tokenPair, h.cookieOpts)
	return res, nil
}

func (h *ConnectHandler) VerifyEmail(ctx context.Context, req *connect.Request[authv1.VerifyEmailRequest]) (*connect.Response[authv1.VerifyEmailResponse], error) {
	if err := h.service.VerifyEmail(ctx, req.Msg.Token); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&authv1.VerifyEmailResponse{
		Message: "email verified",
	}), nil
}

func (h *ConnectHandler) GetOAuthURL(ctx context.Context, req *connect.Request[authv1.GetOAuthURLRequest]) (*connect.Response[authv1.GetOAuthURLResponse], error) {
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	state := hex.EncodeToString(stateBytes)

	url, err := h.oauthService.GetAuthURL(req.Msg.Provider, state)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	res := connect.NewResponse(&authv1.GetOAuthURLResponse{
		Url: url,
	})
	cookie := &http.Cookie{
		Name:     "oauth_state",
		Value:    state,
		Path:     "/",
		Domain:   h.cookieOpts.Domain,
		MaxAge:   600,
		HttpOnly: true,
		Secure:   h.cookieOpts.Secure,
		SameSite: http.SameSiteLaxMode,
	}
	res.Header().Add("Set-Cookie", cookie.String())
	return res, nil
}

func (h *ConnectHandler) OAuthCallback(ctx context.Context, req *connect.Request[authv1.OAuthCallbackRequest]) (*connect.Response[authv1.OAuthCallbackResponse], error) {
	// Validate OAuth state against cookie
	cookieReq := &http.Request{Header: http.Header{"Cookie": req.Header().Values("Cookie")}}
	stateCookie, err := cookieReq.Cookie("oauth_state")
	if err != nil || stateCookie.Value != req.Msg.State {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid oauth state"))
	}

	tokenPair, err := h.oauthService.HandleCallback(ctx, req.Msg.Provider, req.Msg.Code)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	res := connect.NewResponse(&authv1.OAuthCallbackResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt,
	})
	AppendBrowserSessionCookies(res.Header(), tokenPair, h.cookieOpts)
	return res, nil
}

func (h *ConnectHandler) Me(ctx context.Context, req *connect.Request[authv1.MeRequest]) (*connect.Response[authv1.MeResponse], error) {
	user, orgID, err := h.service.Me(ctx, req.Header().Get("Authorization"))
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	return connect.NewResponse(&authv1.MeResponse{
		UserId:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		OrgId:     orgID,
		AvatarUrl: user.AvatarURL,
	}), nil
}

func (h *ConnectHandler) UpdateProfile(ctx context.Context, req *connect.Request[authv1.UpdateProfileRequest]) (*connect.Response[authv1.UpdateProfileResponse], error) {
	msg := req.Msg
	if msg == nil {
		msg = &authv1.UpdateProfileRequest{}
	}

	user, orgID, err := h.service.UpdateProfile(ctx, req.Header().Get("Authorization"), msg.GetName(), msg.GetAvatarUrl())
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	return connect.NewResponse(&authv1.UpdateProfileResponse{
		UserId:    user.ID,
		Email:     user.Email,
		Name:      user.Name,
		OrgId:     orgID,
		AvatarUrl: user.AvatarURL,
	}), nil
}
