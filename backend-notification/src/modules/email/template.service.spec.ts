import { Test, TestingModule } from '@nestjs/testing';
import { TemplateService } from './template.service';

describe('TemplateService', () => {
  let service: TemplateService;

  beforeEach(async () => {
    const module: TestingModule = await Test.createTestingModule({
      providers: [TemplateService],
    }).compile();

    service = module.get<TemplateService>(TemplateService);
  });

  it('should be defined', () => {
    expect(service).toBeDefined();
  });

  describe('render', () => {
    it('should render simple template', () => {
      const template = 'Hello, {{name}}!';
      const result = service.render(template, { name: 'John' });
      expect(result).toBe('Hello, John!');
    });

    it('should render template with multiple variables', () => {
      const template = '{{event_name}} - {{quantity}} tickets - {{total_price}} THB';
      const result = service.render(template, {
        event_name: 'Concert',
        quantity: 2,
        total_price: 3000,
      });
      expect(result).toBe('Concert - 2 tickets - 3000 THB');
    });

    it('should handle missing variables gracefully', () => {
      const template = 'Hello, {{name}}!';
      const result = service.render(template, {});
      expect(result).toBe('Hello, !');
    });
  });

  describe('formatCurrency helper', () => {
    it('should format THB currency', () => {
      const template = 'Total: {{formatCurrency total_price "THB"}}';
      const result = service.render(template, { total_price: 3000 });
      expect(result).toContain('฿');
      expect(result).toContain('3,000');
    });
  });

  describe('validateTemplate', () => {
    it('should return true for valid template', () => {
      const template = 'Hello, {{name}}!';
      expect(service.validateTemplate(template)).toBe(true);
    });

    it('should return true for complex valid template', () => {
      const template = '{{#if name}}Hello, {{name}}!{{/if}}';
      expect(service.validateTemplate(template)).toBe(true);
    });
  });

  describe('uppercase helper', () => {
    it('should uppercase string', () => {
      const template = '{{uppercase zone_name}}';
      const result = service.render(template, { zone_name: 'vip' });
      expect(result).toBe('VIP');
    });
  });
});
