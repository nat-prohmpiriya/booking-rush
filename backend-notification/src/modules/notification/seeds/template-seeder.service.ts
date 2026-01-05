import { Injectable, Logger, OnModuleInit } from '@nestjs/common';
import { NotificationRepository } from '../notification.repository';
import { defaultTemplates } from './templates.seed';
import { TemplateLocale } from '../schemas';

@Injectable()
export class TemplateSeederService implements OnModuleInit {
  private readonly logger = new Logger(TemplateSeederService.name);

  constructor(private readonly repository: NotificationRepository) {}

  async onModuleInit() {
    await this.seedTemplates();
  }

  async seedTemplates(): Promise<void> {
    this.logger.log('Seeding notification templates...');

    for (const template of defaultTemplates) {
      try {
        await this.repository.upsertTemplate(
          template.name,
          template.locale as TemplateLocale,
          {
            subject: template.subject,
            body: template.body,
            description: template.description,
            is_active: true,
            version: 1,
          },
        );
        this.logger.debug(`Seeded template: ${template.name} (${template.locale})`);
      } catch (error) {
        this.logger.error(
          `Failed to seed template ${template.name}: ${error.message}`,
        );
      }
    }

    this.logger.log(`Seeded ${defaultTemplates.length} templates`);
  }
}
