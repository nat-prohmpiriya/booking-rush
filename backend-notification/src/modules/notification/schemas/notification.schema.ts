import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { HydratedDocument } from 'mongoose';

export type NotificationDocument = HydratedDocument<Notification>;

export enum NotificationType {
  BOOKING_CONFIRMATION = 'booking_confirmation',
  E_TICKET = 'e_ticket',
  PAYMENT_RECEIPT = 'payment_receipt',
  BOOKING_EXPIRED = 'booking_expired',
  BOOKING_CANCELLED = 'booking_cancelled',
}

export enum NotificationChannel {
  EMAIL = 'email',
  SMS = 'sms',
  PUSH = 'push',
}

export enum NotificationStatus {
  PENDING = 'pending',
  SENT = 'sent',
  FAILED = 'failed',
  RETRYING = 'retrying',
}

@Schema({ _id: false })
export class NotificationMetadata {
  @Prop()
  event_name?: string;

  @Prop()
  event_id?: string;

  @Prop()
  show_date?: string;

  @Prop()
  zone_name?: string;

  @Prop()
  quantity?: number;

  @Prop()
  unit_price?: number;

  @Prop()
  total_price?: number;

  @Prop()
  currency?: string;

  @Prop()
  confirmation_code?: string;

  @Prop()
  payment_id?: string;

  @Prop()
  payment_method?: string;

  @Prop()
  venue_name?: string;

  @Prop()
  venue_address?: string;

  @Prop()
  qr_code_data?: string;
}

@Schema({
  collection: 'notifications',
  timestamps: { createdAt: 'created_at', updatedAt: 'updated_at' },
})
export class Notification {
  @Prop({ required: true, index: true })
  tenant_id: string;

  @Prop({ required: true, index: true })
  user_id: string;

  @Prop({ required: true, index: true })
  booking_id: string;

  @Prop({
    required: true,
    enum: NotificationType,
    type: String,
  })
  type: NotificationType;

  @Prop({
    required: true,
    enum: NotificationChannel,
    type: String,
    default: NotificationChannel.EMAIL,
  })
  channel: NotificationChannel;

  @Prop({ required: true })
  recipient: string;

  @Prop({ required: true })
  subject: string;

  @Prop({ required: true })
  content: string;

  @Prop({
    required: true,
    enum: NotificationStatus,
    type: String,
    default: NotificationStatus.PENDING,
    index: true,
  })
  status: NotificationStatus;

  @Prop({ default: 0 })
  retry_count: number;

  @Prop()
  sent_at?: Date;

  @Prop()
  error_message?: string;

  @Prop({ type: NotificationMetadata })
  metadata?: NotificationMetadata;

  @Prop()
  idempotency_key?: string;

  created_at: Date;
  updated_at: Date;
}

export const NotificationSchema = SchemaFactory.createForClass(Notification);

// Compound indexes for common queries
NotificationSchema.index({ tenant_id: 1, user_id: 1 });
NotificationSchema.index({ booking_id: 1, type: 1 }, { unique: true }); // Idempotency
NotificationSchema.index({ status: 1, created_at: 1 });

// TTL index - auto delete after 90 days
NotificationSchema.index(
  { created_at: 1 },
  { expireAfterSeconds: 90 * 24 * 60 * 60 },
);
