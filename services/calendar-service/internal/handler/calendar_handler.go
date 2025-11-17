package handler

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/metadata"

	calendarpb "metargb/shared/pb/calendar"
	commonpb "metargb/shared/pb/common"
	"metargb/calendar-service/internal/models"
	"metargb/calendar-service/internal/service"
	"metargb/shared/pkg/jalali"
)

type CalendarHandler struct {
	calendarpb.UnimplementedCalendarServiceServer
	service *service.CalendarService
}

func RegisterCalendarHandler(grpcServer *grpc.Server, svc *service.CalendarService) {
	handler := &CalendarHandler{service: svc}
	calendarpb.RegisterCalendarServiceServer(grpcServer, handler)
}

// GetEvents retrieves events with optional filtering
func (h *CalendarHandler) GetEvents(ctx context.Context, req *calendarpb.GetEventsRequest) (*calendarpb.EventsResponse, error) {
	// Default pagination
	page := int32(1)
	perPage := int32(10)
	if req.Pagination != nil {
		if req.Pagination.Page > 0 {
			page = req.Pagination.Page
		}
		if req.Pagination.PerPage > 0 {
			perPage = req.Pagination.PerPage
		}
	}

	// Get events
	events, total, err := h.service.GetEvents(ctx, req.Type, req.Search, req.Date, req.UserId, page, perPage)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get events: %v", err)
	}

	// Build response
	response := &calendarpb.EventsResponse{
		Events: make([]*calendarpb.EventResponse, 0, len(events)),
		Pagination: &commonpb.PaginationMeta{
			CurrentPage: page,
			PerPage:     perPage,
			Total:       total,
			LastPage:    (total + perPage - 1) / perPage,
		},
	}

	for _, event := range events {
		// Get stats for each event
		stats, _ := h.service.GetEventStats(ctx, event.ID)

		// Get user interaction if user ID provided
		var userInteraction *calendarpb.UserInteraction
		if req.UserId > 0 {
			interaction, _ := h.service.GetUserInteraction(ctx, event.ID, req.UserId)
			if interaction != nil {
				userInteraction = &calendarpb.UserInteraction{
					HasLiked:    interaction.Liked,
					HasDisliked: !interaction.Liked,
				}
			}
		}

		response.Events = append(response.Events, buildEventResponse(event, stats, userInteraction))
	}

	return response, nil
}

// GetEvent retrieves a single event
// NOTE: Laravel auto-increments views on retrieval (CalendarController line 75)
func (h *CalendarHandler) GetEvent(ctx context.Context, req *calendarpb.GetEventRequest) (*calendarpb.EventResponse, error) {
	event, err := h.service.GetEvent(ctx, req.EventId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "event not found: %v", err)
	}

	// Get client IP from metadata
	ipAddress := "unknown"
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if ips := md.Get("x-forwarded-for"); len(ips) > 0 {
			ipAddress = ips[0]
		} else if ips := md.Get("x-real-ip"); len(ips) > 0 {
			ipAddress = ips[0]
		}
	}

	// Auto-increment view count (matching Laravel behavior)
	_ = h.service.IncrementView(ctx, event.ID, ipAddress)

	// Get stats
	stats, _ := h.service.GetEventStats(ctx, event.ID)

	// Get user interaction if user ID provided
	var userInteraction *calendarpb.UserInteraction
	if req.UserId > 0 {
		interaction, _ := h.service.GetUserInteraction(ctx, event.ID, req.UserId)
		if interaction != nil {
			userInteraction = &calendarpb.UserInteraction{
				HasLiked:    interaction.Liked,
				HasDisliked: !interaction.Liked,
			}
		}
	}

	return buildEventResponse(event, stats, userInteraction), nil
}

// FilterByDateRange retrieves events within a date range
// NOTE: Laravel returns simplified format (id, title, starts_at, ends_at, color only)
func (h *CalendarHandler) FilterByDateRange(ctx context.Context, req *calendarpb.FilterByDateRangeRequest) (*calendarpb.SimplifiedEventsResponse, error) {
	events, err := h.service.FilterByDateRange(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to filter events: %v", err)
	}

	response := &calendarpb.SimplifiedEventsResponse{
		Events: make([]*calendarpb.SimplifiedEventResponse, 0, len(events)),
	}

	for _, event := range events {
		simplified := &calendarpb.SimplifiedEventResponse{
			Id:       event.ID,
			Title:    event.Title,
			StartsAt: jalali.CarbonToJalali(event.StartsAt), // Date only Y/m/d
			Color:    event.Color,
		}
		if event.EndsAt != nil {
			simplified.EndsAt = jalali.CarbonToJalali(*event.EndsAt) // Date only Y/m/d
		}
		response.Events = append(response.Events, simplified)
	}

	return response, nil
}

// GetLatestVersion retrieves the latest version title
func (h *CalendarHandler) GetLatestVersion(ctx context.Context, req *calendarpb.GetLatestVersionRequest) (*calendarpb.LatestVersionResponse, error) {
	versionTitle, err := h.service.GetLatestVersionTitle(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get latest version: %v", err)
	}

	return &calendarpb.LatestVersionResponse{
		VersionTitle: versionTitle,
	}, nil
}

// AddInteraction adds or updates a user's interaction with an event
func (h *CalendarHandler) AddInteraction(ctx context.Context, req *calendarpb.AddInteractionRequest) (*calendarpb.EventResponse, error) {
	// Add interaction
	err := h.service.AddInteraction(ctx, req.EventId, req.UserId, req.Liked, "")
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add interaction: %v", err)
	}

	// Return updated event
	return h.GetEvent(ctx, &calendarpb.GetEventRequest{
		EventId: req.EventId,
		UserId:  req.UserId,
	})
}

// Helper function to build event response matching Laravel EventResource format
// Laravel uses conditional fields: events have ends_at, views, likes, etc. Versions only have version_title
func buildEventResponse(event *models.Calendar, stats *models.CalendarStats, userInteraction *calendarpb.UserInteraction) *calendarpb.EventResponse {
	response := &calendarpb.EventResponse{
		Id:          event.ID,
		Title:       event.Title,
		Description: event.Content, // Laravel calls it "description" not "content"
		StartsAt:    jalali.CarbonToJalaliDateTime(event.StartsAt), // Y/m/d H:i format
	}

	// Conditional fields based on is_version
	if !event.IsVersion {
		// Event-specific fields
		if event.EndsAt != nil {
			response.EndsAt = jalali.CarbonToJalaliDateTime(*event.EndsAt) // Y/m/d H:i format
		}
		
		if stats != nil {
			response.Views = stats.ViewsCount
			response.Likes = stats.LikesCount
			response.Dislikes = stats.DislikesCount
		}

		if event.BtnName != nil {
			response.BtnName = *event.BtnName
		}
		if event.BtnLink != nil {
			response.BtnLink = *event.BtnLink
		}
		response.Color = event.Color
		if event.Image != nil {
			response.Image = *event.Image
		}
		
		response.UserInteraction = userInteraction
	} else {
		// Version-specific fields
		if event.VersionTitle != nil {
			response.VersionTitle = *event.VersionTitle
		}
	}

	return response
}

