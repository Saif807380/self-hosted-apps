package controller

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"time"

	"connectrpc.com/connect"
	gamingv1 "github.com/saifkazi/trove/backend/gen/trove/v1/gaming"
	"github.com/saifkazi/trove/backend/gen/trove/v1/gaming/gamingv1connect"
	"github.com/saifkazi/trove/backend/internal/catalog"
	"github.com/saifkazi/trove/backend/internal/dao"
	"github.com/saifkazi/trove/backend/internal/model"
	"github.com/saifkazi/trove/backend/internal/service"
	"github.com/saifkazi/trove/backend/internal/stores"
)

type gamingHandler struct {
	svc *service.GamingService
}

var _ gamingv1connect.GamingServiceHandler = (*gamingHandler)(nil)

func (h *gamingHandler) SearchCatalog(
	ctx context.Context,
	req *connect.Request[gamingv1.SearchCatalogRequest],
) (*connect.Response[gamingv1.SearchCatalogResponse], error) {
	query := strings.TrimSpace(req.Msg.Query)
	if query == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("query required"))
	}
	games, err := h.svc.SearchCatalog(ctx, query)
	if err != nil {
		return nil, connectErr(err)
	}
	out := make([]*gamingv1.CatalogGame, len(games))
	for i, g := range games {
		out[i] = mapCatalogGame(g.IGDBID, g.Title, g.ReleaseYear, g.CoverURL, g.Platforms)
	}
	return connect.NewResponse(&gamingv1.SearchCatalogResponse{Games: out}), nil
}

func (h *gamingHandler) LinkGame(
	ctx context.Context,
	req *connect.Request[gamingv1.LinkGameRequest],
) (*connect.Response[gamingv1.LinkGameResponse], error) {
	id, err := parseUUID(req.Msg.VideoGameId)
	if err != nil || req.Msg.IgdbId <= 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("video_game_id and igdb_id required"))
	}
	if err := h.svc.LinkGame(ctx, id, req.Msg.IgdbId); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamingv1.LinkGameResponse{}), nil
}

func (h *gamingHandler) RefreshGame(
	ctx context.Context,
	req *connect.Request[gamingv1.RefreshGameRequest],
) (*connect.Response[gamingv1.RefreshGameResponse], error) {
	id, err := parseUUID(req.Msg.VideoGameId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.svc.RefreshGame(ctx, id); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamingv1.RefreshGameResponse{}), nil
}

func (h *gamingHandler) SetPsnLink(
	ctx context.Context,
	req *connect.Request[gamingv1.SetPsnLinkRequest],
) (*connect.Response[gamingv1.SetPsnLinkResponse], error) {
	id, err := parseUUID(req.Msg.VideoGameId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	err = h.svc.SetPSNLink(ctx, id, strings.TrimSpace(req.Msg.Url))
	if errors.Is(err, service.ErrInvalidLink) {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamingv1.SetPsnLinkResponse{}), nil
}

func (h *gamingHandler) GetGameInsights(
	ctx context.Context,
	req *connect.Request[gamingv1.GetGameInsightsRequest],
) (*connect.Response[gamingv1.GetGameInsightsResponse], error) {
	id, err := parseUUID(req.Msg.VideoGameId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	in, err := h.svc.GetGameInsights(ctx, id)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(mapInsights(in)), nil
}

func (h *gamingHandler) ListDeals(
	ctx context.Context,
	_ *connect.Request[gamingv1.ListDealsRequest],
) (*connect.Response[gamingv1.ListDealsResponse], error) {
	deals, rate, err := h.svc.ListDeals(ctx)
	if err != nil {
		return nil, connectErr(err)
	}
	out := &gamingv1.ListDealsResponse{UsdToInr: rate}
	for _, d := range deals {
		pb := &gamingv1.Deal{
			VideoGameId: d.VideoGameID.String(), Store: d.Store, Edition: d.Edition, Url: d.URL,
			Currency: d.Currency, BaseMinor: d.BaseMinor, FinalMinor: d.FinalMinor,
			DiscountPct: int32(d.DiscountPct), Lowest: d.Lowest,
		}
		if d.SaleEndsAt != nil {
			end := d.SaleEndsAt.Format(time.RFC3339)
			pb.SaleEndsAt = &end
		}
		out.Deals = append(out.Deals, pb)
	}
	return connect.NewResponse(out), nil
}

func (h *gamingHandler) GetPushConfig(
	_ context.Context,
	_ *connect.Request[gamingv1.GetPushConfigRequest],
) (*connect.Response[gamingv1.GetPushConfigResponse], error) {
	enabled, key := h.svc.PushConfig()
	return connect.NewResponse(&gamingv1.GetPushConfigResponse{Enabled: enabled, PublicKey: key}), nil
}

func (h *gamingHandler) SubscribePush(
	ctx context.Context,
	req *connect.Request[gamingv1.SubscribePushRequest],
) (*connect.Response[gamingv1.SubscribePushResponse], error) {
	m := req.Msg
	u, err := url.Parse(m.Endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" || m.P256Dh == "" || m.Auth == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("an https endpoint, p256dh and auth are required"))
	}
	sub := dao.PushSubscription{Endpoint: m.Endpoint, P256dh: m.P256Dh, Auth: m.Auth}
	if err := h.svc.SubscribePush(ctx, sub, req.Header().Get("User-Agent")); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamingv1.SubscribePushResponse{}), nil
}

func (h *gamingHandler) UnsubscribePush(
	ctx context.Context,
	req *connect.Request[gamingv1.UnsubscribePushRequest],
) (*connect.Response[gamingv1.UnsubscribePushResponse], error) {
	if err := h.svc.UnsubscribePush(ctx, req.Msg.Endpoint); err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamingv1.UnsubscribePushResponse{}), nil
}

func (h *gamingHandler) SendTestPush(
	ctx context.Context,
	_ *connect.Request[gamingv1.SendTestPushRequest],
) (*connect.Response[gamingv1.SendTestPushResponse], error) {
	delivered, err := h.svc.SendTestPush(ctx)
	if err != nil {
		return nil, connectErr(err)
	}
	return connect.NewResponse(&gamingv1.SendTestPushResponse{Delivered: int32(delivered)}), nil
}

func mapInsights(in *service.GameInsights) *gamingv1.GetGameInsightsResponse {
	out := &gamingv1.GetGameInsightsResponse{MatchStatus: "none", UsdToInr: in.USDToINR}
	if m := in.Meta; m != nil {
		out.MatchStatus = m.MatchMethod
		if m.IGDBID != nil {
			year, cover := 0, ""
			if m.ReleaseYear != nil {
				year = int(*m.ReleaseYear)
			}
			if m.CoverURL != nil {
				cover = *m.CoverURL
			}
			out.Match = mapCatalogGame(*m.IGDBID, deref(m.Title), year, cover, intsFrom32(m.Platforms))
		}
		for _, c := range m.Candidates {
			out.Candidates = append(out.Candidates, mapCatalogGame(c.IGDBID, c.Title, c.Year, c.CoverURL, nil))
		}
		if m.ListingsAt != nil {
			at := m.ListingsAt.Format(time.RFC3339)
			out.ListingsCheckedAt = &at
		}
	}

	runs := map[string][]model.PriceRun{}
	for _, r := range in.History {
		runs[r.ListingID.String()] = append(runs[r.ListingID.String()], r)
	}
	for _, l := range in.Listings {
		pb := &gamingv1.Listing{
			Id: l.ID.String(), Store: l.Store, Edition: l.Edition, Title: l.Title, Url: l.URL,
			Currency: l.Currency, IsSearchLink: l.Store == stores.PSN && l.StoreRef == "",
		}
		for _, r := range runs[pb.Id] {
			run := &gamingv1.PriceRun{
				From: r.FirstSeenAt.Format(time.RFC3339), Until: r.LastSeenAt.Format(time.RFC3339),
				BaseMinor: r.BaseMinor, FinalMinor: r.FinalMinor, DiscountPct: int32(r.DiscountPct),
			}
			if r.SaleEndsAt != nil {
				end := r.SaleEndsAt.Format(time.RFC3339)
				run.SaleEndsAt = &end
			}
			pb.History = append(pb.History, run)
			if pb.LowestMinor == nil || r.FinalMinor < *pb.LowestMinor {
				lowest := r.FinalMinor
				pb.LowestMinor = &lowest
			}
		}
		out.Listings = append(out.Listings, pb)
	}
	out.VerdictsEnabled = in.VerdictsEnabled
	out.Verdict = mapVerdict(in.Verdict)
	return out
}

func mapVerdict(v *dao.StoredVerdict) *gamingv1.DeviceVerdict {
	if v == nil {
		return nil
	}
	out := &gamingv1.DeviceVerdict{
		Summary: v.Verdict.Summary, Unavailable: v.Verdict.Unavailable,
		Complete: v.Complete, CreatedAt: v.CreatedAt.Format(time.RFC3339),
	}
	for _, d := range v.Verdict.Devices {
		out.Devices = append(out.Devices, &gamingv1.DeviceRating{
			Device: d.Device, Rating: d.Rating, Expect: d.Expect, Reasons: d.Reasons,
		})
	}
	for _, s := range v.Sources {
		out.Sources = append(out.Sources, &gamingv1.Source{Label: s.Label, Url: s.URL})
	}
	return out
}

// platformNames are the IGDB platforms the UI knows how to label.
var platformNames = map[int]string{
	catalog.PlatformPC:      "pc",
	catalog.PlatformPS4:     "ps4",
	catalog.PlatformPS5:     "ps5",
	catalog.PlatformSwitch:  "switch",
	catalog.PlatformSwitch2: "switch2",
}

func mapCatalogGame(id int64, title string, year int, cover string, platforms []int) *gamingv1.CatalogGame {
	g := &gamingv1.CatalogGame{IgdbId: id, Title: title, Year: int32(year), CoverUrl: cover}
	for _, p := range platforms {
		if name, ok := platformNames[p]; ok {
			g.Platforms = append(g.Platforms, name)
		}
	}
	return g
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func intsFrom32(xs []int32) []int {
	out := make([]int, len(xs))
	for i, x := range xs {
		out[i] = int(x)
	}
	return out
}
