import { Injectable, Logger } from '@nestjs/common';
import * as Handlebars from 'handlebars';

export interface TemplateData {
  // Event info
  event_name?: string;
  event_id?: string;
  show_date?: string;
  zone_name?: string;
  venue_name?: string;
  venue_address?: string;

  // Booking info
  booking_id?: string;
  confirmation_code?: string;
  quantity?: number;
  unit_price?: number;
  total_price?: number;
  currency?: string;

  // Payment info
  payment_id?: string;
  payment_method?: string;

  // QR Code
  qr_code_url?: string;
  qr_code_data?: string;

  // URLs
  rebook_url?: string;
  ticket_url?: string;

  // User info
  user_name?: string;
  user_email?: string;

  // Any additional data
  [key: string]: unknown;
}

@Injectable()
export class TemplateService {
  private readonly logger = new Logger(TemplateService.name);

  constructor() {
    this.registerHelpers();
  }

  private registerHelpers(): void {
    // Format currency
    Handlebars.registerHelper('formatCurrency', (amount: number, currency = 'THB') => {
      if (currency === 'THB') {
        return `฿${amount?.toLocaleString('th-TH') || '0'}`;
      }
      return `${currency} ${amount?.toLocaleString() || '0'}`;
    });

    // Format date
    Handlebars.registerHelper('formatDate', (date: string | Date) => {
      if (!date) return '';
      const d = new Date(date);
      return d.toLocaleDateString('th-TH', {
        weekday: 'long',
        year: 'numeric',
        month: 'long',
        day: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    });

    // Format date short
    Handlebars.registerHelper('formatDateShort', (date: string | Date) => {
      if (!date) return '';
      const d = new Date(date);
      return d.toLocaleDateString('th-TH', {
        year: 'numeric',
        month: 'short',
        day: 'numeric',
      });
    });

    // Uppercase
    Handlebars.registerHelper('uppercase', (str: string) => {
      return str?.toUpperCase() || '';
    });

    // Conditional equal
    Handlebars.registerHelper('eq', (a: unknown, b: unknown) => {
      return a === b;
    });
  }

  /**
   * Render a template string with data
   */
  render(template: string, data: TemplateData): string {
    try {
      const compiledTemplate = Handlebars.compile(template);
      return compiledTemplate(data);
    } catch (error) {
      this.logger.error(`Failed to render template: ${error.message}`);
      throw new Error(`Template rendering failed: ${error.message}`);
    }
  }

  /**
   * Render subject line
   */
  renderSubject(subject: string, data: TemplateData): string {
    return this.render(subject, data);
  }

  /**
   * Render email body
   */
  renderBody(body: string, data: TemplateData): string {
    return this.render(body, data);
  }

  /**
   * Validate template syntax
   */
  validateTemplate(template: string): boolean {
    try {
      Handlebars.compile(template);
      return true;
    } catch {
      return false;
    }
  }
}
