import {
  Controller,
  Get,
  Param,
  Query,
  Headers,
  HttpException,
  HttpStatus,
} from '@nestjs/common';
import { AnalyticsService } from './analytics.service';
import { DateRangeQueryDto } from './dto/analytics.dto';

@Controller('analytics')
export class AnalyticsController {
  constructor(private readonly analyticsService: AnalyticsService) {}

  /**
   * GET /analytics/dashboard
   * Get dashboard overview with key metrics
   */
  @Get('dashboard')
  async getDashboard(@Headers('x-tenant-id') tenantId?: string) {
    const overview = await this.analyticsService.getDashboardOverview(tenantId);
    return {
      success: true,
      data: overview,
    };
  }

  /**
   * GET /analytics/sales
   * Get sales report by period
   */
  @Get('sales')
  async getSalesReport(
    @Query() query: DateRangeQueryDto,
    @Headers('x-tenant-id') tenantId?: string,
  ) {
    const report = await this.analyticsService.getSalesReport(query, tenantId);
    return {
      success: true,
      data: report,
    };
  }

  /**
   * GET /analytics/events/top
   * Get top events by revenue
   */
  @Get('events/top')
  async getTopEvents(
    @Query('limit') limit?: string,
    @Headers('x-tenant-id') tenantId?: string,
  ) {
    const events = await this.analyticsService.getTopEvents(
      limit ? parseInt(limit, 10) : 10,
      tenantId,
    );
    return {
      success: true,
      data: events,
    };
  }

  /**
   * GET /analytics/events/:id
   * Get statistics for a specific event
   */
  @Get('events/:id')
  async getEventStats(
    @Param('id') eventId: string,
    @Headers('x-tenant-id') tenantId?: string,
  ) {
    const stats = await this.analyticsService.getEventStats(eventId, tenantId);
    if (!stats) {
      throw new HttpException('Event not found', HttpStatus.NOT_FOUND);
    }
    return {
      success: true,
      data: stats,
    };
  }

  /**
   * GET /analytics/bookings/recent
   * Get recent bookings
   */
  @Get('bookings/recent')
  async getRecentBookings(
    @Query('limit') limit?: string,
    @Headers('x-tenant-id') tenantId?: string,
  ) {
    const bookings = await this.analyticsService.getRecentBookings(
      limit ? parseInt(limit, 10) : 20,
      tenantId,
    );
    return {
      success: true,
      data: bookings,
    };
  }
}
