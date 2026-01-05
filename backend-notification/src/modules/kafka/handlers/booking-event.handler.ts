import { Injectable, Logger } from '@nestjs/common';
import { NotificationService } from '../../notification/notification.service';
import { NotificationRepository } from '../../notification/notification.repository';
import { EmailService } from '../../email/email.service';
import { QrCodeService } from '../../email/qrcode.service';
import { TemplateService } from '../../email/template.service';
import {
  NotificationType,
  NotificationStatus,
  TemplateLocale,
} from '../../notification/schemas';
import {
  PaymentSuccessEvent,
  BookingExpiredEvent,
  BookingCancelledEvent,
  EventType,
} from '../dto/events.dto';

@Injectable()
export class BookingEventHandler {
  private readonly logger = new Logger(BookingEventHandler.name);

  constructor(
    private readonly notificationService: NotificationService,
    private readonly notificationRepository: NotificationRepository,
    private readonly emailService: EmailService,
    private readonly qrCodeService: QrCodeService,
    private readonly templateService: TemplateService,
  ) {}

  /**
   * Handle payment.success event - send e-ticket + receipt
   */
  async handlePaymentSuccess(event: PaymentSuccessEvent): Promise<void> {
    this.logger.log(
      `Processing payment.success for booking ${event.booking_id}`,
    );

    // Check idempotency - already sent?
    const alreadySent = await this.notificationService.isAlreadySent(
      event.booking_id,
      NotificationType.E_TICKET,
    );

    if (alreadySent) {
      this.logger.warn(
        `E-ticket already sent for booking ${event.booking_id}, skipping`,
      );
      return;
    }

    try {
      // Get e-ticket template
      const template = await this.notificationRepository.findTemplateByName(
        'e_ticket',
        TemplateLocale.TH,
      );

      if (!template) {
        this.logger.error('E-ticket template not found');
        return;
      }

      // Generate QR code
      const qrCodeUrl = await this.qrCodeService.generateTicketQrCode(
        event.booking_id,
        event.confirmation_code,
      );

      // Prepare template data
      const templateData = {
        event_name: event.event_name,
        event_id: event.event_id,
        show_date: event.show_date,
        zone_name: event.zone_name,
        quantity: event.quantity,
        unit_price: event.unit_price,
        total_price: event.amount,
        currency: event.currency,
        confirmation_code: event.confirmation_code,
        payment_id: event.payment_id,
        payment_method: event.payment_method,
        venue_name: event.venue_name || 'TBA',
        venue_address: event.venue_address || '',
        qr_code_url: qrCodeUrl,
        booking_id: event.booking_id,
      };

      // Render email
      const subject = this.templateService.renderSubject(
        template.subject,
        templateData,
      );
      const content = this.templateService.renderBody(
        template.body,
        templateData,
      );

      // Create notification record
      const notification = await this.notificationService.createNotification({
        tenant_id: event.tenant_id,
        user_id: event.user_id,
        booking_id: event.booking_id,
        type: NotificationType.E_TICKET,
        recipient: event.user_email,
        subject,
        content,
        metadata: {
          event_name: event.event_name,
          event_id: event.event_id,
          show_date: event.show_date,
          zone_name: event.zone_name,
          quantity: event.quantity,
          total_price: event.amount,
          currency: event.currency,
          confirmation_code: event.confirmation_code,
          payment_id: event.payment_id,
          qr_code_data: this.qrCodeService.generateTicketData(
            event.booking_id,
            event.confirmation_code,
          ),
        },
      });

      // Send email
      const result = await this.emailService.send({
        to: event.user_email,
        subject,
        html: content,
      });

      // Update notification status
      if (result.success) {
        await this.notificationService.markAsSent(notification._id.toString());
        this.logger.log(`E-ticket sent to ${event.user_email}`);
      } else {
        await this.notificationService.markAsFailed(
          notification._id.toString(),
          result.error || 'Unknown error',
        );
        this.logger.error(`Failed to send e-ticket: ${result.error}`);
      }
    } catch (error) {
      this.logger.error(
        `Error handling payment.success: ${error.message}`,
        error.stack,
      );
    }
  }

  /**
   * Handle booking.expired event
   */
  async handleBookingExpired(event: BookingExpiredEvent): Promise<void> {
    this.logger.log(
      `Processing booking.expired for booking ${event.booking_id}`,
    );

    // Check idempotency
    const alreadySent = await this.notificationService.isAlreadySent(
      event.booking_id,
      NotificationType.BOOKING_EXPIRED,
    );

    if (alreadySent) {
      this.logger.warn(
        `Expiry notice already sent for booking ${event.booking_id}, skipping`,
      );
      return;
    }

    try {
      // Get template
      const template = await this.notificationRepository.findTemplateByName(
        'booking_expired',
        TemplateLocale.TH,
      );

      if (!template) {
        this.logger.error('Booking expired template not found');
        return;
      }

      const templateData = {
        event_name: event.event_name,
        show_date: event.show_date,
        zone_name: event.zone_name,
        quantity: event.quantity,
        rebook_url: `https://bookingrush.com/events/${event.event_id}`,
      };

      const subject = this.templateService.renderSubject(
        template.subject,
        templateData,
      );
      const content = this.templateService.renderBody(
        template.body,
        templateData,
      );

      // Create notification record
      const notification = await this.notificationService.createNotification({
        tenant_id: event.tenant_id,
        user_id: event.user_id,
        booking_id: event.booking_id,
        type: NotificationType.BOOKING_EXPIRED,
        recipient: event.user_email,
        subject,
        content,
        metadata: {
          event_name: event.event_name,
          show_date: event.show_date,
          zone_name: event.zone_name,
          quantity: event.quantity,
        },
      });

      // Send email
      const result = await this.emailService.send({
        to: event.user_email,
        subject,
        html: content,
      });

      if (result.success) {
        await this.notificationService.markAsSent(notification._id.toString());
        this.logger.log(`Expiry notice sent to ${event.user_email}`);
      } else {
        await this.notificationService.markAsFailed(
          notification._id.toString(),
          result.error || 'Unknown error',
        );
      }
    } catch (error) {
      this.logger.error(
        `Error handling booking.expired: ${error.message}`,
        error.stack,
      );
    }
  }

  /**
   * Handle booking.cancelled event
   */
  async handleBookingCancelled(event: BookingCancelledEvent): Promise<void> {
    this.logger.log(
      `Processing booking.cancelled for booking ${event.booking_id}`,
    );

    // Check idempotency
    const alreadySent = await this.notificationService.isAlreadySent(
      event.booking_id,
      NotificationType.BOOKING_CANCELLED,
    );

    if (alreadySent) {
      this.logger.warn(
        `Cancellation notice already sent for booking ${event.booking_id}, skipping`,
      );
      return;
    }

    try {
      // Get template
      const template = await this.notificationRepository.findTemplateByName(
        'booking_cancelled',
        TemplateLocale.TH,
      );

      if (!template) {
        this.logger.error('Booking cancelled template not found');
        return;
      }

      const templateData = {
        event_name: event.event_name,
        confirmation_code: event.confirmation_code,
        refund_amount: event.refund_amount,
      };

      const subject = this.templateService.renderSubject(
        template.subject,
        templateData,
      );
      const content = this.templateService.renderBody(
        template.body,
        templateData,
      );

      // Create notification record
      const notification = await this.notificationService.createNotification({
        tenant_id: event.tenant_id,
        user_id: event.user_id,
        booking_id: event.booking_id,
        type: NotificationType.BOOKING_CANCELLED,
        recipient: event.user_email,
        subject,
        content,
        metadata: {
          event_name: event.event_name,
          confirmation_code: event.confirmation_code,
        },
      });

      // Send email
      const result = await this.emailService.send({
        to: event.user_email,
        subject,
        html: content,
      });

      if (result.success) {
        await this.notificationService.markAsSent(notification._id.toString());
        this.logger.log(`Cancellation notice sent to ${event.user_email}`);
      } else {
        await this.notificationService.markAsFailed(
          notification._id.toString(),
          result.error || 'Unknown error',
        );
      }
    } catch (error) {
      this.logger.error(
        `Error handling booking.cancelled: ${error.message}`,
        error.stack,
      );
    }
  }
}
