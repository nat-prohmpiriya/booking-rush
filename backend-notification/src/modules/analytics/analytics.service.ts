import { Injectable, Logger, OnModuleInit, OnModuleDestroy } from '@nestjs/common';
import { ConfigService } from '@nestjs/config';
import { Pool } from 'pg';
import {
  DashboardOverviewDto,
  SalesReportDto,
  SalesByPeriodDto,
  EventStatsDto,
  TopEventDto,
  RecentBookingDto,
  DateRangeQueryDto,
} from './dto/analytics.dto';

@Injectable()
export class AnalyticsService implements OnModuleInit, OnModuleDestroy {
  private readonly logger = new Logger(AnalyticsService.name);
  private pool: Pool;

  constructor(private readonly configService: ConfigService) {}

  async onModuleInit() {
    const pgConfig = this.configService.get('postgres');
    this.pool = new Pool({
      host: pgConfig.host,
      port: pgConfig.port,
      user: pgConfig.username,
      password: pgConfig.password,
      database: pgConfig.database,
      max: 10,
      idleTimeoutMillis: 30000,
    });

    try {
      const client = await this.pool.connect();
      client.release();
      this.logger.log('PostgreSQL connection established for analytics');
    } catch (error) {
      this.logger.error(`Failed to connect to PostgreSQL: ${error.message}`);
    }
  }

  async onModuleDestroy() {
    if (this.pool) {
      await this.pool.end();
      this.logger.log('PostgreSQL connection closed');
    }
  }

  /**
   * Get dashboard overview with key metrics
   */
  async getDashboardOverview(tenantId?: string): Promise<DashboardOverviewDto> {
    const now = new Date();
    const thisMonthStart = new Date(now.getFullYear(), now.getMonth(), 1);
    const lastMonthStart = new Date(now.getFullYear(), now.getMonth() - 1, 1);
    const lastMonthEnd = new Date(now.getFullYear(), now.getMonth(), 0);

    const tenantFilter = tenantId ? 'AND b.tenant_id = $1' : '';
    const params = tenantId ? [tenantId] : [];

    // Current month stats
    const currentQuery = `
      SELECT
        COALESCE(SUM(b.total_price), 0) as revenue,
        COUNT(DISTINCT b.id) as bookings,
        COALESCE(SUM(b.quantity), 0) as tickets
      FROM bookings b
      WHERE b.status IN ('confirmed', 'completed')
        AND b.created_at >= $${params.length + 1}
        ${tenantFilter}
    `;

    // Last month stats for comparison
    const lastMonthQuery = `
      SELECT
        COALESCE(SUM(b.total_price), 0) as revenue,
        COUNT(DISTINCT b.id) as bookings
      FROM bookings b
      WHERE b.status IN ('confirmed', 'completed')
        AND b.created_at >= $${params.length + 1}
        AND b.created_at < $${params.length + 2}
        ${tenantFilter}
    `;

    // Active events count
    const eventsQuery = `
      SELECT COUNT(DISTINCT e.id) as count
      FROM events e
      WHERE e.status = 'published'
        AND e.end_date >= NOW()
        ${tenantId ? 'AND e.tenant_id = $1' : ''}
    `;

    try {
      const [currentResult, lastMonthResult, eventsResult] = await Promise.all([
        this.pool.query(currentQuery, [...params, thisMonthStart]),
        this.pool.query(lastMonthQuery, [...params, lastMonthStart, lastMonthEnd]),
        this.pool.query(eventsQuery, tenantId ? [tenantId] : []),
      ]);

      const current = currentResult.rows[0];
      const lastMonth = lastMonthResult.rows[0];

      const revenueChange = lastMonth.revenue > 0
        ? ((current.revenue - lastMonth.revenue) / lastMonth.revenue) * 100
        : 0;
      const bookingsChange = lastMonth.bookings > 0
        ? ((current.bookings - lastMonth.bookings) / lastMonth.bookings) * 100
        : 0;

      return {
        total_revenue: parseFloat(current.revenue) || 0,
        total_bookings: parseInt(current.bookings) || 0,
        total_tickets_sold: parseInt(current.tickets) || 0,
        active_events: parseInt(eventsResult.rows[0].count) || 0,
        revenue_change_percent: Math.round(revenueChange * 100) / 100,
        bookings_change_percent: Math.round(bookingsChange * 100) / 100,
      };
    } catch (error) {
      this.logger.error(`Error getting dashboard overview: ${error.message}`);
      return {
        total_revenue: 0,
        total_bookings: 0,
        total_tickets_sold: 0,
        active_events: 0,
        revenue_change_percent: 0,
        bookings_change_percent: 0,
      };
    }
  }

  /**
   * Get sales report by period
   */
  async getSalesReport(
    query: DateRangeQueryDto,
    tenantId?: string,
  ): Promise<SalesReportDto> {
    const period = query.period || 'day';
    const endDate = query.end_date ? new Date(query.end_date) : new Date();
    const startDate = query.start_date
      ? new Date(query.start_date)
      : new Date(endDate.getTime() - 30 * 24 * 60 * 60 * 1000); // 30 days ago

    const dateFormat = period === 'month'
      ? "TO_CHAR(b.created_at, 'YYYY-MM')"
      : period === 'week'
      ? "TO_CHAR(DATE_TRUNC('week', b.created_at), 'YYYY-MM-DD')"
      : "TO_CHAR(b.created_at, 'YYYY-MM-DD')";

    const tenantFilter = tenantId ? 'AND b.tenant_id = $3' : '';
    const params: any[] = [startDate, endDate];
    if (tenantId) params.push(tenantId);

    const salesQuery = `
      SELECT
        ${dateFormat} as period,
        COALESCE(SUM(b.total_price), 0) as revenue,
        COUNT(DISTINCT b.id) as bookings,
        COALESCE(SUM(b.quantity), 0) as tickets
      FROM bookings b
      WHERE b.status IN ('confirmed', 'completed')
        AND b.created_at >= $1
        AND b.created_at <= $2
        ${tenantFilter}
      GROUP BY ${dateFormat}
      ORDER BY period ASC
    `;

    try {
      const result = await this.pool.query(salesQuery, params);

      const data: SalesByPeriodDto[] = result.rows.map((row: Record<string, string>) => ({
        period: row.period,
        revenue: parseFloat(row.revenue) || 0,
        bookings: parseInt(row.bookings) || 0,
        tickets_sold: parseInt(row.tickets) || 0,
      }));

      const totals = data.reduce(
        (acc, item) => ({
          revenue: acc.revenue + item.revenue,
          bookings: acc.bookings + item.bookings,
          tickets: acc.tickets + item.tickets_sold,
        }),
        { revenue: 0, bookings: 0, tickets: 0 },
      );

      return {
        start_date: startDate.toISOString().split('T')[0],
        end_date: endDate.toISOString().split('T')[0],
        total_revenue: totals.revenue,
        total_bookings: totals.bookings,
        total_tickets_sold: totals.tickets,
        data,
      };
    } catch (error) {
      this.logger.error(`Error getting sales report: ${error.message}`);
      return {
        start_date: startDate.toISOString().split('T')[0],
        end_date: endDate.toISOString().split('T')[0],
        total_revenue: 0,
        total_bookings: 0,
        total_tickets_sold: 0,
        data: [],
      };
    }
  }

  /**
   * Get statistics for a specific event
   */
  async getEventStats(eventId: string, tenantId?: string): Promise<EventStatsDto | null> {
    const tenantFilter = tenantId ? 'AND e.tenant_id = $2' : '';
    const params: any[] = [eventId];
    if (tenantId) params.push(tenantId);

    const eventQuery = `
      SELECT
        e.id as event_id,
        e.name as event_name,
        COUNT(DISTINCT s.id) as total_shows,
        COUNT(DISTINCT b.id) as total_bookings,
        COALESCE(SUM(b.quantity), 0) as total_tickets,
        COALESCE(SUM(b.total_price), 0) as total_revenue
      FROM events e
      LEFT JOIN shows s ON s.event_id = e.id
      LEFT JOIN bookings b ON b.event_id = e.id AND b.status IN ('confirmed', 'completed')
      WHERE e.id = $1
        ${tenantFilter}
      GROUP BY e.id, e.name
    `;

    const showsQuery = `
      SELECT
        s.id as show_id,
        s.show_date,
        COUNT(DISTINCT b.id) as total_bookings,
        COALESCE(SUM(b.quantity), 0) as total_tickets,
        COALESCE(SUM(b.total_price), 0) as revenue
      FROM shows s
      LEFT JOIN bookings b ON b.show_id = s.id AND b.status IN ('confirmed', 'completed')
      WHERE s.event_id = $1
      GROUP BY s.id, s.show_date
      ORDER BY s.show_date ASC
    `;

    const zonesQuery = `
      SELECT
        z.id as zone_id,
        z.name as zone_name,
        z.total_seats,
        z.available_seats,
        (z.total_seats - z.available_seats) as sold_seats,
        COALESCE(SUM(b.total_price), 0) as revenue,
        CASE WHEN z.total_seats > 0
          THEN ROUND(((z.total_seats - z.available_seats)::numeric / z.total_seats) * 100, 2)
          ELSE 0
        END as occupancy_rate
      FROM seat_zones z
      LEFT JOIN bookings b ON b.zone_id = z.id AND b.status IN ('confirmed', 'completed')
      WHERE z.event_id = $1
      GROUP BY z.id, z.name, z.total_seats, z.available_seats
      ORDER BY z.name ASC
    `;

    try {
      const [eventResult, showsResult, zonesResult] = await Promise.all([
        this.pool.query(eventQuery, params),
        this.pool.query(showsQuery, [eventId]),
        this.pool.query(zonesQuery, [eventId]),
      ]);

      if (eventResult.rows.length === 0) {
        return null;
      }

      const event = eventResult.rows[0];
      const avgPrice = event.total_tickets > 0
        ? event.total_revenue / event.total_tickets
        : 0;

      return {
        event_id: event.event_id,
        event_name: event.event_name,
        total_shows: parseInt(event.total_shows) || 0,
        total_bookings: parseInt(event.total_bookings) || 0,
        total_tickets_sold: parseInt(event.total_tickets) || 0,
        total_revenue: parseFloat(event.total_revenue) || 0,
        average_ticket_price: Math.round(avgPrice * 100) / 100,
        shows: showsResult.rows.map((row: Record<string, string>) => ({
          show_id: row.show_id,
          show_date: row.show_date,
          total_bookings: parseInt(row.total_bookings) || 0,
          total_tickets: parseInt(row.total_tickets) || 0,
          revenue: parseFloat(row.revenue) || 0,
          zones: [], // Simplified - zones are at event level
        })),
      };
    } catch (error) {
      this.logger.error(`Error getting event stats: ${error.message}`);
      return null;
    }
  }

  /**
   * Get top events by revenue
   */
  async getTopEvents(limit: number = 10, tenantId?: string): Promise<TopEventDto[]> {
    const tenantFilter = tenantId ? 'AND e.tenant_id = $2' : '';
    const params: any[] = [limit];
    if (tenantId) params.push(tenantId);

    const query = `
      SELECT
        e.id as event_id,
        e.name as event_name,
        COALESCE(SUM(b.total_price), 0) as total_revenue,
        COUNT(DISTINCT b.id) as total_bookings,
        COALESCE(SUM(b.quantity), 0) as total_tickets
      FROM events e
      LEFT JOIN bookings b ON b.event_id = e.id AND b.status IN ('confirmed', 'completed')
      WHERE 1=1 ${tenantFilter}
      GROUP BY e.id, e.name
      ORDER BY total_revenue DESC
      LIMIT $1
    `;

    try {
      const result = await this.pool.query(query, params);
      return result.rows.map((row: Record<string, string>) => ({
        event_id: row.event_id,
        event_name: row.event_name,
        total_revenue: parseFloat(row.total_revenue) || 0,
        total_bookings: parseInt(row.total_bookings) || 0,
        total_tickets: parseInt(row.total_tickets) || 0,
      }));
    } catch (error) {
      this.logger.error(`Error getting top events: ${error.message}`);
      return [];
    }
  }

  /**
   * Get recent bookings
   */
  async getRecentBookings(
    limit: number = 20,
    tenantId?: string,
  ): Promise<RecentBookingDto[]> {
    const tenantFilter = tenantId ? 'AND b.tenant_id = $2' : '';
    const params: any[] = [limit];
    if (tenantId) params.push(tenantId);

    const query = `
      SELECT
        b.id as booking_id,
        e.name as event_name,
        z.name as zone_name,
        b.quantity,
        b.total_price,
        b.status,
        b.created_at
      FROM bookings b
      LEFT JOIN events e ON e.id = b.event_id
      LEFT JOIN seat_zones z ON z.id = b.zone_id
      WHERE 1=1 ${tenantFilter}
      ORDER BY b.created_at DESC
      LIMIT $1
    `;

    try {
      const result = await this.pool.query(query, params);
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      return result.rows.map((row: any) => ({
        booking_id: row.booking_id,
        event_name: row.event_name || 'Unknown',
        zone_name: row.zone_name || 'Unknown',
        quantity: parseInt(row.quantity) || 0,
        total_price: parseFloat(row.total_price) || 0,
        status: row.status,
        created_at: row.created_at?.toISOString() || '',
      }));
    } catch (error) {
      this.logger.error(`Error getting recent bookings: ${error.message}`);
      return [];
    }
  }

  /**
   * Check if database is healthy
   */
  async isHealthy(): Promise<boolean> {
    try {
      await this.pool.query('SELECT 1');
      return true;
    } catch {
      return false;
    }
  }
}
