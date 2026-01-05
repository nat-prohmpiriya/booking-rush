import { Module } from '@nestjs/common';
import { MongooseModule } from '@nestjs/mongoose';
import {
  Notification,
  NotificationSchema,
  NotificationTemplate,
  NotificationTemplateSchema,
} from './schemas';
import { NotificationService } from './notification.service';
import { NotificationRepository } from './notification.repository';
import { TemplateSeederService } from './seeds/template-seeder.service';

@Module({
  imports: [
    MongooseModule.forFeature([
      { name: Notification.name, schema: NotificationSchema },
      { name: NotificationTemplate.name, schema: NotificationTemplateSchema },
    ]),
  ],
  providers: [NotificationService, NotificationRepository, TemplateSeederService],
  exports: [NotificationService, NotificationRepository],
})
export class NotificationModule {}
