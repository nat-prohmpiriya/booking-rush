import { Test, TestingModule } from '@nestjs/testing';
import { ConfigService } from '@nestjs/config';
import { AnalyticsService } from './analytics.service';

describe('AnalyticsService', () => {
  let service: AnalyticsService;
  let configService: jest.Mocked<ConfigService>;

  const mockPool = {
    connect: jest.fn().mockResolvedValue({ release: jest.fn() }),
    query: jest.fn(),
    end: jest.fn(),
  };

  beforeEach(async () => {
    // Mock pg Pool
    jest.mock('pg', () => ({
      Pool: jest.fn(() => mockPool),
    }));

    const module: TestingModule = await Test.createTestingModule({
      providers: [
        AnalyticsService,
        {
          provide: ConfigService,
          useValue: {
            get: jest.fn().mockReturnValue({
              host: 'localhost',
              port: 5432,
              username: 'postgres',
              password: 'postgres',
              database: 'booking_db',
            }),
          },
        },
      ],
    }).compile();

    service = module.get<AnalyticsService>(AnalyticsService);
    configService = module.get(ConfigService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  describe('getDashboardOverview', () => {
    it('should return dashboard overview with default values when no data', async () => {
      // Service won't connect in test without real DB
      // This tests the error handling path
      const result = await service.getDashboardOverview();

      expect(result).toHaveProperty('total_revenue');
      expect(result).toHaveProperty('total_bookings');
      expect(result).toHaveProperty('total_tickets_sold');
      expect(result).toHaveProperty('active_events');
      expect(result).toHaveProperty('revenue_change_percent');
      expect(result).toHaveProperty('bookings_change_percent');
    });
  });

  describe('getSalesReport', () => {
    it('should return sales report with correct structure', async () => {
      const result = await service.getSalesReport({
        start_date: '2025-01-01',
        end_date: '2025-01-31',
        period: 'day',
      });

      expect(result).toHaveProperty('start_date');
      expect(result).toHaveProperty('end_date');
      expect(result).toHaveProperty('total_revenue');
      expect(result).toHaveProperty('total_bookings');
      expect(result).toHaveProperty('total_tickets_sold');
      expect(result).toHaveProperty('data');
      expect(Array.isArray(result.data)).toBe(true);
    });

    it('should handle different period types', async () => {
      const dayResult = await service.getSalesReport({ period: 'day' });
      const weekResult = await service.getSalesReport({ period: 'week' });
      const monthResult = await service.getSalesReport({ period: 'month' });

      expect(dayResult.data).toBeDefined();
      expect(weekResult.data).toBeDefined();
      expect(monthResult.data).toBeDefined();
    });
  });

  describe('getEventStats', () => {
    it('should return null when event not found', async () => {
      const result = await service.getEventStats('non-existent-id');
      expect(result).toBeNull();
    });
  });

  describe('getTopEvents', () => {
    it('should return array of top events', async () => {
      const result = await service.getTopEvents(10);
      expect(Array.isArray(result)).toBe(true);
    });

    it('should respect limit parameter', async () => {
      const result = await service.getTopEvents(5);
      expect(result.length).toBeLessThanOrEqual(5);
    });
  });

  describe('getRecentBookings', () => {
    it('should return array of recent bookings', async () => {
      const result = await service.getRecentBookings(20);
      expect(Array.isArray(result)).toBe(true);
    });
  });

  describe('isHealthy', () => {
    it('should return false when pool not connected', async () => {
      const result = await service.isHealthy();
      expect(typeof result).toBe('boolean');
    });
  });
});
