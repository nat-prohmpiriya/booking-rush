import { apiClient } from "./client"
import type {
  EventResponse,
  EventListResponse,
  EventListFilter,
  ShowResponse,
  ShowListResponse,
  ShowZoneResponse,
  ShowZoneListResponse,
} from "./types"

export const eventsApi = {
  async list(filter?: EventListFilter): Promise<EventListResponse> {
    const params = new URLSearchParams()
    if (filter?.status) params.append("status", filter.status)
    if (filter?.venue_id) params.append("venue_id", filter.venue_id)
    if (filter?.search) params.append("search", filter.search)
    if (filter?.limit) params.append("limit", filter.limit.toString())
    if (filter?.offset) params.append("offset", filter.offset.toString())

    const queryString = params.toString()
    const endpoint = queryString ? `/events?${queryString}` : "/events"
    return apiClient.get<EventListResponse>(endpoint)
  },

  async getBySlug(slug: string): Promise<EventResponse> {
    return apiClient.get<EventResponse>(`/events/${slug}`)
  },

  async getById(id: string): Promise<EventResponse> {
    return apiClient.get<EventResponse>(`/events/id/${id}`)
  },
}

export const showsApi = {
  async listByEvent(eventSlug: string, limit?: number, offset?: number): Promise<ShowListResponse> {
    const params = new URLSearchParams()
    if (limit) params.append("limit", limit.toString())
    if (offset) params.append("offset", offset.toString())

    const queryString = params.toString()
    const endpoint = queryString
      ? `/events/${eventSlug}/shows?${queryString}`
      : `/events/${eventSlug}/shows`
    return apiClient.get<ShowListResponse>(endpoint)
  },

  async getById(showId: string): Promise<ShowResponse> {
    return apiClient.get<ShowResponse>(`/shows/${showId}`)
  },
}

export const zonesApi = {
  async listByShow(showId: string, limit?: number, offset?: number): Promise<ShowZoneListResponse> {
    const params = new URLSearchParams()
    if (limit) params.append("limit", limit.toString())
    if (offset) params.append("offset", offset.toString())

    const queryString = params.toString()
    const endpoint = queryString
      ? `/shows/${showId}/zones?${queryString}`
      : `/shows/${showId}/zones`
    return apiClient.get<ShowZoneListResponse>(endpoint)
  },

  async getById(zoneId: string): Promise<ShowZoneResponse> {
    return apiClient.get<ShowZoneResponse>(`/zones/${zoneId}`)
  },
}
