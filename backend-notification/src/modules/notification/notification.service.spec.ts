import { Test, TestingModule } from '@nestjs/testing';
import { NotificationService } from './notification.service';
import { NotificationRepository } from './notification.repository';
import {
  NotificationType,
  NotificationStatus,
  NotificationChannel,
} from './schemas';

describe('NotificationService', () => {
  let service: NotificationService;
  let repository: jest.Mocked<NotificationRepository>;

  const mockNotification = {
    _id: 'notification-id-1',
    tenant_id: 'tenant-1',
    user_id: 'user-1',
    booking_id: 'booking-1',
    type: NotificationType.E_TICKET,
    channel: NotificationChannel.EMAIL,
    recipient: 'test@example.com',
    subject: 'Your E-Ticket',
    content: '<html>...</html>',
    status: NotificationStatus.PENDING,
    retry_count: 0,
    metadata: {
      event_name: 'Concert',
      quantity: 2,
    },
    created_at: new Date(),
    updated_at: new Date(),
  };

  beforeEach(async () => {
    const mockRepository = {
      create: jest.fn(),
      findById: jest.fn(),
      findByBookingIdAndType: jest.fn(),
      findByUserId: jest.fn(),
      updateStatus: jest.fn(),
      incrementRetryCount: jest.fn(),
      findPendingNotifications: jest.fn(),
      findTemplateByName: jest.fn(),
    };

    const module: TestingModule = await Test.createTestingModule({
      providers: [
        NotificationService,
        { provide: NotificationRepository, useValue: mockRepository },
      ],
    }).compile();

    service = module.get<NotificationService>(NotificationService);
    repository = module.get(NotificationRepository);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  describe('createNotification', () => {
    it('should create a new notification', async () => {
      repository.findByBookingIdAndType.mockResolvedValue(null);
      repository.create.mockResolvedValue(mockNotification as any);

      const result = await service.createNotification({
        tenant_id: 'tenant-1',
        user_id: 'user-1',
        booking_id: 'booking-1',
        type: NotificationType.E_TICKET,
        recipient: 'test@example.com',
        subject: 'Your E-Ticket',
        content: '<html>...</html>',
      });

      expect(result).toEqual(mockNotification);
      expect(repository.create).toHaveBeenCalled();
    });

    it('should return existing notification if already exists (idempotency)', async () => {
      repository.findByBookingIdAndType.mockResolvedValue(
        mockNotification as any,
      );

      const result = await service.createNotification({
        tenant_id: 'tenant-1',
        user_id: 'user-1',
        booking_id: 'booking-1',
        type: NotificationType.E_TICKET,
        recipient: 'test@example.com',
        subject: 'Your E-Ticket',
        content: '<html>...</html>',
      });

      expect(result).toEqual(mockNotification);
      expect(repository.create).not.toHaveBeenCalled();
    });
  });

  describe('markAsSent', () => {
    it('should update status to SENT', async () => {
      const sentNotification = {
        ...mockNotification,
        status: NotificationStatus.SENT,
      };
      repository.updateStatus.mockResolvedValue(sentNotification as any);

      const result = await service.markAsSent('notification-id-1');

      expect(result).not.toBeNull();
      expect(result!.status).toBe(NotificationStatus.SENT);
      expect(repository.updateStatus).toHaveBeenCalledWith(
        'notification-id-1',
        NotificationStatus.SENT,
      );
    });
  });

  describe('markAsFailed', () => {
    it('should update status to FAILED with error message', async () => {
      const failedNotification = {
        ...mockNotification,
        status: NotificationStatus.FAILED,
        error_message: 'SMTP error',
      };
      repository.updateStatus.mockResolvedValue(failedNotification as any);

      const result = await service.markAsFailed(
        'notification-id-1',
        'SMTP error',
      );

      expect(result).not.toBeNull();
      expect(result!.status).toBe(NotificationStatus.FAILED);
      expect(repository.updateStatus).toHaveBeenCalledWith(
        'notification-id-1',
        NotificationStatus.FAILED,
        'SMTP error',
      );
    });
  });

  describe('isAlreadySent', () => {
    it('should return true if notification was sent', async () => {
      const sentNotification = {
        ...mockNotification,
        status: NotificationStatus.SENT,
      };
      repository.findByBookingIdAndType.mockResolvedValue(
        sentNotification as any,
      );

      const result = await service.isAlreadySent(
        'booking-1',
        NotificationType.E_TICKET,
      );

      expect(result).toBe(true);
    });

    it('should return false if notification was not sent', async () => {
      repository.findByBookingIdAndType.mockResolvedValue(
        mockNotification as any,
      );

      const result = await service.isAlreadySent(
        'booking-1',
        NotificationType.E_TICKET,
      );

      expect(result).toBe(false);
    });

    it('should return false if notification does not exist', async () => {
      repository.findByBookingIdAndType.mockResolvedValue(null);

      const result = await service.isAlreadySent(
        'booking-1',
        NotificationType.E_TICKET,
      );

      expect(result).toBe(false);
    });
  });
});
