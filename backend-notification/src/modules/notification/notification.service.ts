import { Injectable, Logger } from '@nestjs/common';
import {
  NotificationRepository,
  CreateNotificationDto,
} from './notification.repository';
import {
  NotificationDocument,
  NotificationType,
  NotificationStatus,
  TemplateLocale,
} from './schemas';

@Injectable()
export class NotificationService {
  private readonly logger = new Logger(NotificationService.name);

  constructor(private readonly repository: NotificationRepository) {}

  async createNotification(
    dto: CreateNotificationDto,
  ): Promise<NotificationDocument> {
    // Check for idempotency - prevent duplicate notifications
    const existing = await this.repository.findByBookingIdAndType(
      dto.booking_id,
      dto.type,
    );

    if (existing) {
      this.logger.warn(
        `Notification already exists for booking ${dto.booking_id} type ${dto.type}`,
      );
      return existing;
    }

    return this.repository.create(dto);
  }

  async markAsSent(id: string): Promise<NotificationDocument | null> {
    return this.repository.updateStatus(id, NotificationStatus.SENT);
  }

  async markAsFailed(
    id: string,
    errorMessage: string,
  ): Promise<NotificationDocument | null> {
    return this.repository.updateStatus(
      id,
      NotificationStatus.FAILED,
      errorMessage,
    );
  }

  async retryNotification(id: string): Promise<NotificationDocument | null> {
    return this.repository.incrementRetryCount(id);
  }

  async getNotificationsByUser(
    userId: string,
    limit = 20,
    skip = 0,
  ): Promise<NotificationDocument[]> {
    return this.repository.findByUserId(userId, limit, skip);
  }

  async getNotificationById(id: string): Promise<NotificationDocument | null> {
    return this.repository.findById(id);
  }

  async getTemplate(name: string, locale: TemplateLocale = TemplateLocale.TH) {
    return this.repository.findTemplateByName(name, locale);
  }

  async getPendingNotifications(limit = 100): Promise<NotificationDocument[]> {
    return this.repository.findPendingNotifications(limit);
  }

  // Check if notification already sent (for idempotency)
  async isAlreadySent(
    bookingId: string,
    type: NotificationType,
  ): Promise<boolean> {
    const existing = await this.repository.findByBookingIdAndType(
      bookingId,
      type,
    );
    return existing?.status === NotificationStatus.SENT;
  }
}
