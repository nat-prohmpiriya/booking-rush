// Dashboard Overview Response
export class DashboardOverviewDto {
  total_revenue: number;
  total_bookings: number;
  total_tickets_sold: number;
  active_events: number;
  revenue_change_percent: number;
  bookings_change_percent: number;
}

// Sales by Period
export class SalesByPeriodDto {
  period: string; // date or week or month
  revenue: number;
  bookings: number;
  tickets_sold: number;
}

export class SalesReportDto {
  start_date: string;
  end_date: string;
  total_revenue: number;
  total_bookings: number;
  total_tickets_sold: number;
  data: SalesByPeriodDto[];
}

// Event Statistics
export class ZoneStatsDto {
  zone_id: string;
  zone_name: string;
  total_seats: number;
  sold_seats: number;
  available_seats: number;
  revenue: number;
  occupancy_rate: number;
}

export class ShowStatsDto {
  show_id: string;
  show_date: string;
  total_bookings: number;
  total_tickets: number;
  revenue: number;
  zones: ZoneStatsDto[];
}

export class EventStatsDto {
  event_id: string;
  event_name: string;
  total_shows: number;
  total_bookings: number;
  total_tickets_sold: number;
  total_revenue: number;
  average_ticket_price: number;
  shows: ShowStatsDto[];
}

// Top Events
export class TopEventDto {
  event_id: string;
  event_name: string;
  total_revenue: number;
  total_bookings: number;
  total_tickets: number;
}

// Recent Bookings
export class RecentBookingDto {
  booking_id: string;
  event_name: string;
  zone_name: string;
  quantity: number;
  total_price: number;
  status: string;
  created_at: string;
}

// Query Parameters
export class DateRangeQueryDto {
  start_date?: string;
  end_date?: string;
  period?: 'day' | 'week' | 'month';
}

export class PaginationQueryDto {
  page?: number;
  limit?: number;
}
