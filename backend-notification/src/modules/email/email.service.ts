import { Injectable, Logger } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Resend } from 'resend';
import { TemplateService, TemplateData } from './template.service';
import { QrCodeService } from './qrcode.service';

export interface SendEmailOptions {
  to: string;
  subject: string;
  html: string;
  text?: string;
  replyTo?: string;
  tags?: Array<{ name: string; value: string }>;
}

export interface SendEmailResult {
  success: boolean;
  messageId?: string;
  error?: string;
}

export interface SendTemplatedEmailOptions {
  to: string;
  templateSubject: string;
  templateBody: string;
  data: TemplateData;
  includeQrCode?: boolean;
  bookingId?: string;
  confirmationCode?: string;
}

@Injectable()
export class EmailService {
  private readonly logger = new Logger(EmailService.name);
  private readonly resend: Resend;
  private readonly fromEmail: string;
  private readonly maxRetries = 3;
  private readonly baseDelay = 1000; // 1 second

  constructor(
    private readonly configService: ConfigService,
    private readonly templateService: TemplateService,
    private readonly qrCodeService: QrCodeService,
  ) {
    const apiKey = this.configService.get<string>('resend.apiKey');
    this.fromEmail = this.configService.get<string>('resend.fromEmail') || 'Booking Rush <onboarding@resend.dev>';

    if (apiKey) {
      this.resend = new Resend(apiKey);
      this.logger.log('Resend email service initialized');
    } else {
      this.logger.warn('Resend API key not configured - emails will be logged only');
    }
  }

  /**
   * Send email with retry logic
   */
  async send(options: SendEmailOptions): Promise<SendEmailResult> {
    let lastError: Error | null = null;

    for (let attempt = 1; attempt <= this.maxRetries; attempt++) {
      try {
        const result = await this.sendOnce(options);
        if (result.success) {
          return result;
        }
        lastError = new Error(result.error);
      } catch (error) {
        lastError = error;
        this.logger.warn(
          `Email send attempt ${attempt}/${this.maxRetries} failed: ${error.message}`,
        );
      }

      // Don't wait after last attempt
      if (attempt < this.maxRetries) {
        const delay = this.calculateBackoff(attempt);
        this.logger.debug(`Waiting ${delay}ms before retry...`);
        await this.sleep(delay);
      }
    }

    this.logger.error(
      `Failed to send email after ${this.maxRetries} attempts: ${lastError?.message}`,
    );

    return {
      success: false,
      error: lastError?.message || 'Unknown error',
    };
  }

  /**
   * Send email once (no retry)
   */
  private async sendOnce(options: SendEmailOptions): Promise<SendEmailResult> {
    // If no API key, just log
    if (!this.resend) {
      this.logger.log(`[DRY RUN] Would send email to: ${options.to}`);
      this.logger.debug(`Subject: ${options.subject}`);
      return {
        success: true,
        messageId: `dry-run-${Date.now()}`,
      };
    }

    try {
      const response = await this.resend.emails.send({
        from: this.fromEmail,
        to: options.to,
        subject: options.subject,
        html: options.html,
        text: options.text,
        replyTo: options.replyTo,
        tags: options.tags,
      });

      if (response.error) {
        return {
          success: false,
          error: response.error.message,
        };
      }

      this.logger.log(`Email sent successfully to ${options.to}, ID: ${response.data?.id}`);

      return {
        success: true,
        messageId: response.data?.id,
      };
    } catch (error) {
      return {
        success: false,
        error: error.message,
      };
    }
  }

  /**
   * Send templated email with optional QR code
   */
  async sendTemplated(options: SendTemplatedEmailOptions): Promise<SendEmailResult> {
    let data = { ...options.data };

    // Generate QR code if needed
    if (options.includeQrCode && options.bookingId && options.confirmationCode) {
      try {
        const qrCodeUrl = await this.qrCodeService.generateTicketQrCode(
          options.bookingId,
          options.confirmationCode,
        );
        data.qr_code_url = qrCodeUrl;
      } catch (error) {
        this.logger.warn(`Failed to generate QR code: ${error.message}`);
      }
    }

    // Render templates
    const subject = this.templateService.renderSubject(options.templateSubject, data);
    const html = this.templateService.renderBody(options.templateBody, data);

    return this.send({
      to: options.to,
      subject,
      html,
    });
  }

  /**
   * Calculate exponential backoff delay
   */
  private calculateBackoff(attempt: number): number {
    // Exponential backoff: 1s, 2s, 4s
    const delay = this.baseDelay * Math.pow(2, attempt - 1);
    // Add jitter (0-500ms) to prevent thundering herd
    const jitter = Math.random() * 500;
    return delay + jitter;
  }

  /**
   * Sleep helper
   */
  private sleep(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms));
  }

  /**
   * Check if email service is configured
   */
  isConfigured(): boolean {
    return !!this.resend;
  }
}
