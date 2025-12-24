import { Prop, Schema, SchemaFactory } from '@nestjs/mongoose';
import { HydratedDocument } from 'mongoose';

export type NotificationTemplateDocument =
  HydratedDocument<NotificationTemplate>;

export enum TemplateLocale {
  TH = 'th',
  EN = 'en',
}

@Schema({
  collection: 'notification_templates',
  timestamps: { createdAt: 'created_at', updatedAt: 'updated_at' },
})
export class NotificationTemplate {
  @Prop({ required: true, index: true })
  name: string;

  @Prop({ required: true })
  subject: string;

  @Prop({ required: true })
  body: string;

  @Prop({
    required: true,
    enum: TemplateLocale,
    type: String,
    default: TemplateLocale.TH,
  })
  locale: TemplateLocale;

  @Prop({ default: true })
  is_active: boolean;

  @Prop({ default: 1 })
  version: number;

  @Prop()
  description?: string;

  created_at: Date;
  updated_at: Date;
}

export const NotificationTemplateSchema =
  SchemaFactory.createForClass(NotificationTemplate);

// Compound index for unique template per name and locale
NotificationTemplateSchema.index({ name: 1, locale: 1 }, { unique: true });

// Index for active templates
NotificationTemplateSchema.index({ is_active: 1, name: 1 });
