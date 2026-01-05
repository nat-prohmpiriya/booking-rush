import { Test, TestingModule } from '@nestjs/testing';
import { QrCodeService } from './qrcode.service';

describe('QrCodeService', () => {
  let service: QrCodeService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [QrCodeService],
    }).compile();

    service = module.get<QrCodeService>(QrCodeService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  describe('generateDataUrl', () => {
    it('should generate a data URL', async () => {
      const result = await service.generateDataUrl('test-data');
      expect(result).toMatch(/^data:image\/png;base64,/);
    });

    it('should generate different QR codes for different data', async () => {
      const result1 = await service.generateDataUrl('data1');
      const result2 = await service.generateDataUrl('data2');
      expect(result1).not.toBe(result2);
    });
  });

  describe('generateBuffer', () => {
    it('should generate a Buffer', async () => {
      const result = await service.generateBuffer('test-data');
      expect(result).toBeInstanceOf(Buffer);
      expect(result.length).toBeGreaterThan(0);
    });
  });

  describe('generateTicketData', () => {
    it('should generate ticket data in correct format', () => {
      const result = service.generateTicketData('booking-123', 'CONF-ABC');
      expect(result).toBe('BOOKING:booking-123:CONF-ABC');
    });
  });

  describe('generateTicketQrCode', () => {
    it('should generate QR code for ticket', async () => {
      const result = await service.generateTicketQrCode('booking-123', 'CONF-ABC');
      expect(result).toMatch(/^data:image\/png;base64,/);
    });
  });
});
