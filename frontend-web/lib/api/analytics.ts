import { apiClient } from './client'
import type {
  ApiResponse,
  DashboardOverviewResponse,
  SalesReportResponse,
  SalesReportFilter,
  TopEventResponse,
  RecentBookingResponse,
  EventStatsResponse,
} from './types'

// Analytics API base URL (notification service)
const ANALYTICS_BASE_URL = process.env.NEXT_PUBLIC_ANALYTICS_URL || 'http://localhost:8085/api/v1'

/**
 * Analytics API client
 * Connects to notification service analytics endpoints
 */
export const analyticsApi = {
  /**
   * Get dashboard overview with key metrics
   */
  async getDashboard(): Promise<DashboardOverviewResponse> {
    const response = await apiClient.get<ApiResponse<DashboardOverviewResponse>>(
      '/analytics/dashboard',
      { baseURL: ANALYTICS_BASE_URL }
    )
    return response.data!
  },

  /**
   * Get sales report by period
   */
  async getSalesReport(filter?: SalesReportFilter): Promise<SalesReportResponse> {
    const params = new URLSearchParams()
    if (filter?.start_date) params.append('start_date', filter.start_date)
    if (filter?.end_date) params.append('end_date', filter.end_date)
    if (filter?.period) params.append('period', filter.period)

    const response = await apiClient.get<ApiResponse<SalesReportResponse>>(
      `/analytics/sales?${params.toString()}`,
      { baseURL: ANALYTICS_BASE_URL }
    )
    return response.data!
  },

  /**
   * Get top events by revenue
   */
  async getTopEvents(limit: number = 10): Promise<TopEventResponse[]> {
    const response = await apiClient.get<ApiResponse<TopEventResponse[]>>(
      `/analytics/events/top?limit=${limit}`,
      { baseURL: ANALYTICS_BASE_URL }
    )
    return response.data!
  },

  /**
   * Get statistics for a specific event
   */
  async getEventStats(eventId: string): Promise<EventStatsResponse> {
    const response = await apiClient.get<ApiResponse<EventStatsResponse>>(
      `/analytics/events/${eventId}`,
      { baseURL: ANALYTICS_BASE_URL }
    )
    return response.data!
  },

  /**
   * Get recent bookings
   */
  async getRecentBookings(limit: number = 20): Promise<RecentBookingResponse[]> {
    const response = await apiClient.get<ApiResponse<RecentBookingResponse[]>>(
      `/analytics/bookings/recent?limit=${limit}`,
      { baseURL: ANALYTICS_BASE_URL }
    )
    return response.data!
  },
}
