'use client'

import { useState, useEffect, useCallback } from 'react'
import { analyticsApi } from '@/lib/api/analytics'
import type {
  DashboardOverviewResponse,
  SalesReportResponse,
  SalesReportFilter,
  TopEventResponse,
  RecentBookingResponse,
  EventStatsResponse,
} from '@/lib/api/types'

/**
 * Hook for fetching dashboard overview data
 */
export function useDashboardOverview() {
  const [data, setData] = useState<DashboardOverviewResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await analyticsApi.getDashboard()
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch dashboard data')
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}

/**
 * Hook for fetching sales report data
 */
export function useSalesReport(filter?: SalesReportFilter) {
  const [data, setData] = useState<SalesReportResponse | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await analyticsApi.getSalesReport(filter)
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch sales report')
    } finally {
      setIsLoading(false)
    }
  }, [filter?.start_date, filter?.end_date, filter?.period])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}

/**
 * Hook for fetching top events
 */
export function useTopEvents(limit: number = 10) {
  const [data, setData] = useState<TopEventResponse[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await analyticsApi.getTopEvents(limit)
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch top events')
    } finally {
      setIsLoading(false)
    }
  }, [limit])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}

/**
 * Hook for fetching recent bookings
 */
export function useRecentBookings(limit: number = 20) {
  const [data, setData] = useState<RecentBookingResponse[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const result = await analyticsApi.getRecentBookings(limit)
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch recent bookings')
    } finally {
      setIsLoading(false)
    }
  }, [limit])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}

/**
 * Hook for fetching event statistics
 */
export function useEventStats(eventId: string | null) {
  const [data, setData] = useState<EventStatsResponse | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchData = useCallback(async () => {
    if (!eventId) return

    try {
      setIsLoading(true)
      setError(null)
      const result = await analyticsApi.getEventStats(eventId)
      setData(result)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch event stats')
    } finally {
      setIsLoading(false)
    }
  }, [eventId])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}
