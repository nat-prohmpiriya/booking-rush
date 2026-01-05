import { Injectable, Logger } from '@nestjs/common';
import * as QRCode from 'qrcode';

export interface QrCodeOptions {
  width?: number;
  margin?: number;
  color?: {
    dark?: string;
    light?: string;
  };
}

@Injectable()
export class QrCodeService {
  private readonly logger = new Logger(QrCodeService.name);

  private readonly defaultOptions: QrCodeOptions = {
    width: 200,
    margin: 2,
    color: {
      dark: '#000000',
      light: '#ffffff',
    },
  };

  /**
   * Generate QR code as base64 data URL
   * Can be used directly in <img src="...">
   */
  async generateDataUrl(
    data: string,
    options?: QrCodeOptions,
  ): Promise<string> {
    try {
      const opts = { ...this.defaultOptions, ...options };
      const dataUrl = await QRCode.toDataURL(data, {
        width: opts.width,
        margin: opts.margin,
        color: opts.color,
      });
      return dataUrl;
    } catch (error) {
      this.logger.error(`Failed to generate QR code: ${error.message}`);
      throw new Error(`QR code generation failed: ${error.message}`);
    }
  }

  /**
   * Generate QR code as Buffer (for file saving or attachment)
   */
  async generateBuffer(
    data: string,
    options?: QrCodeOptions,
  ): Promise<Buffer> {
    try {
      const opts = { ...this.defaultOptions, ...options };
      const buffer = await QRCode.toBuffer(data, {
        width: opts.width,
        margin: opts.margin,
        color: opts.color,
      });
      return buffer;
    } catch (error) {
      this.logger.error(`Failed to generate QR code buffer: ${error.message}`);
      throw new Error(`QR code generation failed: ${error.message}`);
    }
  }

  /**
   * Generate ticket QR code data
   * Format: BOOKING:{booking_id}:{confirmation_code}
   */
  generateTicketData(bookingId: string, confirmationCode: string): string {
    return `BOOKING:${bookingId}:${confirmationCode}`;
  }

  /**
   * Generate complete ticket QR code as data URL
   */
  async generateTicketQrCode(
    bookingId: string,
    confirmationCode: string,
    options?: QrCodeOptions,
  ): Promise<string> {
    const data = this.generateTicketData(bookingId, confirmationCode);
    return this.generateDataUrl(data, options);
  }
}
