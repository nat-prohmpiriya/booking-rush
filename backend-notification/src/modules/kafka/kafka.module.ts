import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { KafkaConsumerService } from './kafka-consumer.service';
import { BookingEventHandler } from './handlers/booking-event.handler';
import { NotificationModule } from '../notification/notification.module';
import { EmailModule } from '../email/email.module';

@Module({
  imports: [ConfigModule, NotificationModule, EmailModule],
  providers: [KafkaConsumerService, BookingEventHandler],
  exports: [KafkaConsumerService],
})
export class KafkaModule {}
