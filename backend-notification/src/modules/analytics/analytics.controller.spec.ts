import { Test, TestingModule } from '@nestjs/testing';
import { HttpException } from '@nestjs/common';
import { AnalyticsController } from './analytics.controller';
import { AnalyticsService } from './analytics.service';

describe('AnalyticsController', () => {
  let controller: AnalyticsController;
  let analyticsService: jest.Mocked<AnalyticsService>;

  const mockDashboard = {
    total_revenue: 150000,
    total_bookings: 100,
    total_tickets_sold: 250,
    active_events: 5,
    revenue_change_percent: 15.5,
    bookings_change_percent: 10.2,
  };

  const mockSalesReport = {
    start_date: '2025-01-01',
    end_date: '2025-01-31',
    total_revenue: 150000,
    total_bookings: 100,
    total_tickets_sold: 250,
    data: [
      { period: '2025-01-01', revenue: 5000, bookings: 10, tickets_sold: 25 },
    ],
  };

  const mockEventStats = {
    event_id: 'event-123',
    event_name: 'Test Concert',
    total_shows: 3,
    total_bookings: 50,
    total_tickets_sold: 100,
    total_revenue: 50000,
    average_ticket_price: 500,
    shows: [],
  };

  const mockTopEvents = [
    {
      event_id: 'event-1',
      event_name: 'Concert A',
      total_revenue: 100000,
      total_bookings: 50,
      total_tickets: 100,
    },
  ];

  const mockRecentBookings = [
    {
      booking_id: 'booking-1',
      event_name: 'Concert A',
      zone_name: 'VIP',
      quantity: 2,
      total_price: 3000,
      status: 'confirmed',
      created_at: '2025-01-15T10:00:00Z',
    },
  ];

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      controllers: [AnalyticsController],
      providers: [
        {
          provide: AnalyticsService,
          useValue: {
            getDashboardOverview: jest.fn().mockResolvedValue(mockDashboard),
            getSalesReport: jest.fn().mockResolvedValue(mockSalesReport),
            getEventStats: jest.fn().mockResolvedValue(mockEventStats),
            getTopEvents: jest.fn().mockResolvedValue(mockTopEvents),
            getRecentBookings: jest.fn().mockResolvedValue(mockRecentBookings),
          },
        },
      ],
    }).compile();

    controller = module.get<AnalyticsController>(AnalyticsController);
    analyticsService = module.get(AnalyticsService);
  });

  it('should be defined', () => {
    expect(controller).toBeDefined();
  });

  describe('getDashboard', () => {
    it('should return dashboard overview', async () => {
      const result = await controller.getDashboard();

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockDashboard);
      expect(analyticsService.getDashboardOverview).toHaveBeenCalled();
    });

    it('should pass tenant_id if provided', async () => {
      await controller.getDashboard('tenant-123');

      expect(analyticsService.getDashboardOverview).toHaveBeenCalledWith(
        'tenant-123',
      );
    });
  });

  describe('getSalesReport', () => {
    it('should return sales report', async () => {
      const query = { start_date: '2025-01-01', period: 'day' as const };
      const result = await controller.getSalesReport(query);

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockSalesReport);
    });

    it('should pass tenant_id if provided', async () => {
      await controller.getSalesReport({}, 'tenant-123');

      expect(analyticsService.getSalesReport).toHaveBeenCalledWith(
        {},
        'tenant-123',
      );
    });
  });

  describe('getTopEvents', () => {
    it('should return top events', async () => {
      const result = await controller.getTopEvents();

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockTopEvents);
    });

    it('should parse limit parameter', async () => {
      await controller.getTopEvents('5');

      expect(analyticsService.getTopEvents).toHaveBeenCalledWith(
        5,
        undefined,
      );
    });
  });

  describe('getEventStats', () => {
    it('should return event statistics', async () => {
      const result = await controller.getEventStats('event-123');

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockEventStats);
    });

    it('should throw 404 when event not found', async () => {
      analyticsService.getEventStats.mockResolvedValue(null);

      await expect(controller.getEventStats('non-existent')).rejects.toThrow(
        HttpException,
      );
    });
  });

  describe('getRecentBookings', () => {
    it('should return recent bookings', async () => {
      const result = await controller.getRecentBookings();

      expect(result.success).toBe(true);
      expect(result.data).toEqual(mockRecentBookings);
    });

    it('should parse limit parameter', async () => {
      await controller.getRecentBookings('10');

      expect(analyticsService.getRecentBookings).toHaveBeenCalledWith(
        10,
        undefined,
      );
    });
  });
});
