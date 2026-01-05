import { Test, TestingModule } from '@nestjs/testing';
import { ConfigService } from '@nestjs/config';
import { EmailService } from './email.service';
import { TemplateService } from './template.service';
import { QrCodeService } from './qrcode.service';

describe('EmailService', () => {
  let service: EmailService;
  let templateService: jest.Mocked<TemplateService>;
  let qrCodeService: jest.Mocked<QrCodeService>;

  beforeEach(async () => {
    const mockConfigService = {
      get: jest.fn((key: string) => {
        const config: Record<string, string> = {
          'resend.apiKey': '', // Empty = dry run mode
          'resend.fromEmail': 'test@example.com',
        };
        return config[key];
      }),
    };

    const mockTemplateService = {
      renderSubject: jest.fn((template, data) => `Rendered: ${template}`),
      renderBody: jest.fn((template, data) => `<html>${template}</html>`),
    };

    const mockQrCodeService = {
      generateTicketQrCode: jest.fn().mockResolvedValue('data:image/png;base64,xxx'),
    };

    const module: TestingModule = await Test.createTestingModule({
      providers: [
        EmailService,
        { provide: ConfigService, useValue: mockConfigService },
        { provide: TemplateService, useValue: mockTemplateService },
        { provide: QrCodeService, useValue: mockQrCodeService },
      ],
    }).compile();

    service = module.get<EmailService>(EmailService);
    templateService = module.get(TemplateService);
    qrCodeService = module.get(QrCodeService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  describe('send', () => {
    it('should send email in dry run mode when API key not configured', async () => {
      const result = await service.send({
        to: 'test@example.com',
        subject: 'Test',
        html: '<p>Hello</p>',
      });

      expect(result.success).toBe(true);
      expect(result.messageId).toMatch(/^dry-run-/);
    });
  });

  describe('sendTemplated', () => {
    it('should render template and send email', async () => {
      const result = await service.sendTemplated({
        to: 'test@example.com',
        templateSubject: 'Your E-Ticket - {{event_name}}',
        templateBody: '<html>{{event_name}}</html>',
        data: { event_name: 'Concert' },
      });

      expect(result.success).toBe(true);
      expect(templateService.renderSubject).toHaveBeenCalled();
      expect(templateService.renderBody).toHaveBeenCalled();
    });

    it('should include QR code when requested', async () => {
      const result = await service.sendTemplated({
        to: 'test@example.com',
        templateSubject: 'Your E-Ticket',
        templateBody: '<html>QR: {{qr_code_url}}</html>',
        data: { event_name: 'Concert' },
        includeQrCode: true,
        bookingId: 'booking-123',
        confirmationCode: 'CONF-ABC',
      });

      expect(result.success).toBe(true);
      expect(qrCodeService.generateTicketQrCode).toHaveBeenCalledWith(
        'booking-123',
        'CONF-ABC',
      );
    });
  });

  describe('isConfigured', () => {
    it('should return false when API key not configured', () => {
      expect(service.isConfigured()).toBe(false);
    });
  });
});
