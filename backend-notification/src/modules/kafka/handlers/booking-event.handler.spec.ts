import { Test, TestingModule } from '@nestjs/testing';
import { BookingEventHandler } from './booking-event.handler';
import { NotificationService } from '../../notification/notification.service';
import { NotificationRepository } from '../../notification/notification.repository';
import { EmailService } from '../../email/email.service';
import { QrCodeService } from '../../email/qrcode.service';
import { TemplateService } from '../../email/template.service';
import { NotificationType } from '../../notification/schemas';
import { PaymentSuccessEvent, BookingExpiredEvent } from '../dto/events.dto';

describe('BookingEventHandler', () => {
  let handler: BookingEventHandler;
  let notificationService: jest.Mocked<NotificationService>;
  let notificationRepository: jest.Mocked<NotificationRepository>;
  let emailService: jest.Mocked<EmailService>;
  let qrCodeService: jest.Mocked<QrCodeService>;
  let templateService: jest.Mocked<TemplateService>;

  const mockNotification = {
    _id: { toString: () => 'notification-id' },
    booking_id: 'booking-123',
    type: NotificationType.E_TICKET,
    status: 'pending',
  };

  const mockTemplate = {
    name: 'e_ticket',
    subject: 'Your E-Ticket - {{event_name}}',
    body: '<html>{{event_name}}</html>',
  };

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [
        BookingEventHandler,
        {
          provide: NotificationService,
          useValue: {
            isAlreadySent: jest.fn().mockResolvedValue(false),
            createNotification: jest.fn().mockResolvedValue(mockNotification),
            markAsSent: jest.fn().mockResolvedValue(mockNotification),
            markAsFailed: jest.fn().mockResolvedValue(mockNotification),
          },
        },
        {
          provide: NotificationRepository,
          useValue: {
            findTemplateByName: jest.fn().mockResolvedValue(mockTemplate),
          },
        },
        {
          provide: EmailService,
          useValue: {
            send: jest.fn().mockResolvedValue({ success: true, messageId: 'msg-123' }),
          },
        },
        {
          provide: QrCodeService,
          useValue: {
            generateTicketQrCode: jest.fn().mockResolvedValue('data:image/png;base64,xxx'),
            generateTicketData: jest.fn().mockReturnValue('BOOKING:booking-123:CONF-ABC'),
          },
        },
        {
          provide: TemplateService,
          useValue: {
            renderSubject: jest.fn().mockReturnValue('Your E-Ticket - Concert'),
            renderBody: jest.fn().mockReturnValue('<html>Concert</html>'),
          },
        },
      ],
    }).compile();

    handler = module.get<BookingEventHandler>(BookingEventHandler);
    notificationService = module.get(NotificationService);
    notificationRepository = module.get(NotificationRepository);
    emailService = module.get(EmailService);
    qrCodeService = module.get(QrCodeService);
    templateService = module.get(TemplateService);
  });

  it('should be defined', () => {
    expect(handler).toBeDefined();
  });

  describe('handlePaymentSuccess', () => {
    const paymentEvent: PaymentSuccessEvent = {
      event_type: 'payment.success',
      timestamp: new Date().toISOString(),
      booking_id: 'booking-123',
      tenant_id: 'tenant-1',
      user_id: 'user-1',
      user_email: 'test@example.com',
      payment_id: 'payment-123',
      amount: 3000,
      currency: 'THB',
      payment_method: 'credit_card',
      event_id: 'event-1',
      event_name: 'Concert',
      show_id: 'show-1',
      show_date: '2025-01-01T19:00:00Z',
      zone_id: 'zone-1',
      zone_name: 'VIP',
      quantity: 2,
      unit_price: 1500,
      confirmation_code: 'CONF-ABC',
    };

    it('should send e-ticket for payment success', async () => {
      await handler.handlePaymentSuccess(paymentEvent);

      expect(notificationService.isAlreadySent).toHaveBeenCalledWith(
        'booking-123',
        NotificationType.E_TICKET,
      );
      expect(qrCodeService.generateTicketQrCode).toHaveBeenCalled();
      expect(notificationService.createNotification).toHaveBeenCalled();
      expect(emailService.send).toHaveBeenCalled();
      expect(notificationService.markAsSent).toHaveBeenCalled();
    });

    it('should skip if already sent (idempotency)', async () => {
      notificationService.isAlreadySent.mockResolvedValue(true);

      await handler.handlePaymentSuccess(paymentEvent);

      expect(notificationService.createNotification).not.toHaveBeenCalled();
      expect(emailService.send).not.toHaveBeenCalled();
    });

    it('should handle email failure', async () => {
      emailService.send.mockResolvedValue({ success: false, error: 'SMTP error' });

      await handler.handlePaymentSuccess(paymentEvent);

      expect(notificationService.markAsFailed).toHaveBeenCalledWith(
        'notification-id',
        'SMTP error',
      );
    });
  });

  describe('handleBookingExpired', () => {
    const expiredEvent: BookingExpiredEvent = {
      event_type: 'booking.expired',
      timestamp: new Date().toISOString(),
      booking_id: 'booking-456',
      tenant_id: 'tenant-1',
      user_id: 'user-1',
      user_email: 'test@example.com',
      event_id: 'event-1',
      event_name: 'Concert',
      show_date: '2025-01-01T19:00:00Z',
      zone_name: 'VIP',
      quantity: 2,
      expired_at: new Date().toISOString(),
    };

    it('should send expiry notice', async () => {
      notificationRepository.findTemplateByName.mockResolvedValue({
        ...mockTemplate,
        name: 'booking_expired',
      } as any);

      await handler.handleBookingExpired(expiredEvent);

      expect(notificationService.isAlreadySent).toHaveBeenCalledWith(
        'booking-456',
        NotificationType.BOOKING_EXPIRED,
      );
      expect(notificationService.createNotification).toHaveBeenCalled();
      expect(emailService.send).toHaveBeenCalled();
    });

    it('should skip if already sent', async () => {
      notificationService.isAlreadySent.mockResolvedValue(true);

      await handler.handleBookingExpired(expiredEvent);

      expect(notificationService.createNotification).not.toHaveBeenCalled();
    });
  });
});
