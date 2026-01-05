import { Module } from '@nestjs/common';
import { ConfigModule } from '@nestjs/config';
import { EmailService } from './email.service';
import { TemplateService } from './template.service';
import { QrCodeService } from './qrcode.service';

@Module({
  imports: [ConfigModule],
  providers: [EmailService, TemplateService, QrCodeService],
  exports: [EmailService, TemplateService, QrCodeService],
})
export class EmailModule {}
