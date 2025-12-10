package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/domain"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/dto"
	"github.com/prohmpiriya/booking-rush-10k-rps/backend-ticket/internal/repository"
)

// ShowService errors
var (
	ErrShowNotFound = errors.New("show not found")
)

// showService implements the ShowService interface
type showService struct {
	showRepo  repository.ShowRepository
	eventRepo repository.EventRepository
}

// NewShowService creates a new ShowService
func NewShowService(showRepo repository.ShowRepository, eventRepo repository.EventRepository) ShowService {
	return &showService{
		showRepo:  showRepo,
		eventRepo: eventRepo,
	}
}

// CreateShow creates a new show for an event
func (s *showService) CreateShow(ctx context.Context, req *dto.CreateShowRequest) (*domain.Show, error) {
	// Validate request
	if valid, msg := req.Validate(); !valid {
		return nil, errors.New(msg)
	}

	// Verify event exists
	event, err := s.eventRepo.GetByID(ctx, req.EventID)
	if err != nil {
		return nil, err
	}
	if event == nil {
		return nil, ErrEventNotFound
	}

	// Parse date and times from string
	showDate, err := time.Parse("2006-01-02", req.ShowDate)
	if err != nil {
		return nil, errors.New("invalid show_date format, expected YYYY-MM-DD")
	}
	startTime, err := time.Parse("15:04:05Z07:00", req.StartTime)
	if err != nil {
		// Try without timezone
		startTime, err = time.Parse("15:04:05", req.StartTime)
		if err != nil {
			return nil, errors.New("invalid start_time format")
		}
	}
	var endTime time.Time
	if req.EndTime != "" {
		endTime, err = time.Parse("15:04:05Z07:00", req.EndTime)
		if err != nil {
			endTime, _ = time.Parse("15:04:05", req.EndTime)
		}
	}
	var doorsOpenAt *time.Time
	if req.DoorsOpenAt != "" {
		t, err := time.Parse("15:04:05Z07:00", req.DoorsOpenAt)
		if err != nil {
			t, _ = time.Parse("15:04:05", req.DoorsOpenAt)
		}
		doorsOpenAt = &t
	}

	// Create show
	now := time.Now()
	show := &domain.Show{
		ID:          uuid.New().String(),
		EventID:     req.EventID,
		Name:        req.Name,
		ShowDate:    showDate,
		StartTime:   startTime,
		EndTime:     endTime,
		DoorsOpenAt: doorsOpenAt,
		Status:      domain.ShowStatusScheduled,
		SaleStartAt: req.SaleStartAt,
		SaleEndAt:   req.SaleEndAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.showRepo.Create(ctx, show); err != nil {
		return nil, err
	}

	return show, nil
}

// GetShowByID retrieves a show by ID
func (s *showService) GetShowByID(ctx context.Context, id string) (*domain.Show, error) {
	show, err := s.showRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if show == nil {
		return nil, ErrShowNotFound
	}
	return show, nil
}

// ListShowsByEvent lists shows for an event
func (s *showService) ListShowsByEvent(ctx context.Context, eventID string, filter *dto.ShowListFilter) ([]*domain.Show, int, error) {
	filter.SetDefaults()

	// Verify event exists
	event, err := s.eventRepo.GetByID(ctx, eventID)
	if err != nil {
		return nil, 0, err
	}
	if event == nil {
		return nil, 0, ErrEventNotFound
	}

	return s.showRepo.GetByEventID(ctx, eventID, filter.Limit, filter.Offset)
}

// UpdateShow updates a show
func (s *showService) UpdateShow(ctx context.Context, id string, req *dto.UpdateShowRequest) (*domain.Show, error) {
	// Validate request
	if valid, msg := req.Validate(); !valid {
		return nil, errors.New(msg)
	}

	// Get existing show
	show, err := s.showRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if show == nil {
		return nil, ErrShowNotFound
	}

	// Update fields
	if req.Name != "" {
		show.Name = req.Name
	}
	if req.ShowDate != "" {
		showDate, err := time.Parse("2006-01-02", req.ShowDate)
		if err == nil {
			show.ShowDate = showDate
		}
	}
	if req.StartTime != "" {
		startTime, err := time.Parse("15:04:05Z07:00", req.StartTime)
		if err != nil {
			startTime, _ = time.Parse("15:04:05", req.StartTime)
		}
		show.StartTime = startTime
	}
	if req.EndTime != "" {
		endTime, err := time.Parse("15:04:05Z07:00", req.EndTime)
		if err != nil {
			endTime, _ = time.Parse("15:04:05", req.EndTime)
		}
		show.EndTime = endTime
	}
	if req.DoorsOpenAt != "" {
		t, err := time.Parse("15:04:05Z07:00", req.DoorsOpenAt)
		if err != nil {
			t, _ = time.Parse("15:04:05", req.DoorsOpenAt)
		}
		show.DoorsOpenAt = &t
	}
	if req.Status != "" {
		show.Status = req.Status
	}
	if req.SaleStartAt != nil {
		show.SaleStartAt = req.SaleStartAt
	}
	if req.SaleEndAt != nil {
		show.SaleEndAt = req.SaleEndAt
	}

	if err := s.showRepo.Update(ctx, show); err != nil {
		return nil, err
	}

	return show, nil
}

// DeleteShow soft deletes a show
func (s *showService) DeleteShow(ctx context.Context, id string) error {
	// Check if show exists
	show, err := s.showRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if show == nil {
		return ErrShowNotFound
	}

	return s.showRepo.Delete(ctx, id)
}
