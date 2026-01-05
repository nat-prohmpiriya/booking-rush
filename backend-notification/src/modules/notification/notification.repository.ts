import { Injectable } from '@nestjs/common';
import { InjectModel } from '@nestjs/mongoose';
import { Model } from 'mongoose';
import {
  Notification,
  NotificationDocument,
  NotificationType,
  NotificationStatus,
  NotificationTemplate,
  NotificationTemplateDocument,
  TemplateLocale,
} from './schemas';

export interface CreateNotificationDto {
  tenant_id: string;
  user_id: string;
  booking_id: string;
  type: NotificationType;
  recipient: string;
  subject: string;
  content: string;
  metadata?: Notification['metadata'];
  idempotency_key?: string;
}

@Injectable()
export class NotificationRepository {
  constructor(
    @InjectModel(Notification.name)
    private notificationModel: Model<NotificationDocument>,
    @InjectModel(NotificationTemplate.name)
    private templateModel: Model<NotificationTemplateDocument>,
  ) {}

  // Notification methods
  async create(dto: CreateNotificationDto): Promise<NotificationDocument> {
    const notification = new this.notificationModel(dto);
    return notification.save();
  }

  async findById(id: string): Promise<NotificationDocument | null> {
    return this.notificationModel.findById(id).exec();
  }

  async findByBookingIdAndType(
    bookingId: string,
    type: NotificationType,
  ): Promise<NotificationDocument | null> {
    return this.notificationModel
      .findOne({ booking_id: bookingId, type })
      .exec();
  }

  async findByUserId(
    userId: string,
    limit = 20,
    skip = 0,
  ): Promise<NotificationDocument[]> {
    return this.notificationModel
      .find({ user_id: userId })
      .sort({ created_at: -1 })
      .skip(skip)
      .limit(limit)
      .exec();
  }

  async updateStatus(
    id: string,
    status: NotificationStatus,
    errorMessage?: string,
  ): Promise<NotificationDocument | null> {
    const update: Partial<Notification> = { status };

    if (status === NotificationStatus.SENT) {
      update.sent_at = new Date();
    }

    if (errorMessage) {
      update.error_message = errorMessage;
    }

    return this.notificationModel
      .findByIdAndUpdate(id, { $set: update }, { new: true })
      .exec();
  }

  async incrementRetryCount(id: string): Promise<NotificationDocument | null> {
    return this.notificationModel
      .findByIdAndUpdate(
        id,
        {
          $inc: { retry_count: 1 },
          $set: { status: NotificationStatus.RETRYING },
        },
        { new: true },
      )
      .exec();
  }

  async findPendingNotifications(
    limit = 100,
  ): Promise<NotificationDocument[]> {
    return this.notificationModel
      .find({
        status: { $in: [NotificationStatus.PENDING, NotificationStatus.RETRYING] },
        retry_count: { $lt: 3 },
      })
      .sort({ created_at: 1 })
      .limit(limit)
      .exec();
  }

  // Template methods
  async findTemplateByName(
    name: string,
    locale: TemplateLocale = TemplateLocale.TH,
  ): Promise<NotificationTemplateDocument | null> {
    return this.templateModel
      .findOne({ name, locale, is_active: true })
      .exec();
  }

  async createTemplate(
    template: Partial<NotificationTemplate>,
  ): Promise<NotificationTemplateDocument> {
    const newTemplate = new this.templateModel(template);
    return newTemplate.save();
  }

  async findAllTemplates(): Promise<NotificationTemplateDocument[]> {
    return this.templateModel.find({ is_active: true }).exec();
  }

  async upsertTemplate(
    name: string,
    locale: TemplateLocale,
    data: Partial<NotificationTemplate>,
  ): Promise<NotificationTemplateDocument> {
    return this.templateModel
      .findOneAndUpdate(
        { name, locale },
        { $set: { ...data, name, locale } },
        { upsert: true, new: true },
      )
      .exec();
  }
}
