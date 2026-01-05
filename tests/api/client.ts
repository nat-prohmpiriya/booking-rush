import axios, { AxiosInstance, AxiosError } from 'axios'

const API_BASE_URL = process.env.API_BASE_URL || 'http://localhost:8080/api/v1'

/**
 * Create axios instance for API testing
 */
export function createClient(token?: string): AxiosInstance {
  const client = axios.create({
    baseURL: API_BASE_URL,
    timeout: 10000,
    headers: {
      'Content-Type': 'application/json',
      ...(token && { Authorization: `Bearer ${token}` }),
    },
  })

  // Log requests in debug mode
  if (process.env.DEBUG) {
    client.interceptors.request.use((config) => {
      console.log(`→ ${config.method?.toUpperCase()} ${config.url}`)
      return config
    })

    client.interceptors.response.use(
      (response) => {
        console.log(`← ${response.status} ${response.config.url}`)
        return response
      },
      (error: AxiosError) => {
        console.log(`← ${error.response?.status || 'ERR'} ${error.config?.url}`)
        return Promise.reject(error)
      }
    )
  }

  return client
}

/**
 * API helper class with common methods
 */
export class ApiClient {
  private client: AxiosInstance
  private token?: string

  constructor(token?: string) {
    this.token = token
    this.client = createClient(token)
  }

  setToken(token: string) {
    this.token = token
    this.client = createClient(token)
  }

  // Auth endpoints
  async login(email: string, password: string) {
    const res = await this.client.post('/auth/login', { email, password })
    return res.data
  }

  async register(email: string, password: string, name: string) {
    const res = await this.client.post('/auth/register', { email, password, name })
    return res.data
  }

  async getCurrentUser() {
    const res = await this.client.get('/auth/me')
    return res.data
  }

  // Events endpoints
  async listEvents(params?: { limit?: number; offset?: number }) {
    const res = await this.client.get('/events', { params })
    return res.data
  }

  async getEvent(eventId: string) {
    const res = await this.client.get(`/events/${eventId}`)
    return res.data
  }

  async getEventShows(eventId: string) {
    const res = await this.client.get(`/events/${eventId}/shows`)
    return res.data
  }

  async getShowZones(showId: string) {
    const res = await this.client.get(`/shows/${showId}/zones`)
    return res.data
  }

  // Booking endpoints
  async reserveSeats(data: {
    event_id: string
    zone_id: string
    show_id: string
    quantity: number
    unit_price: number
  }) {
    const idempotencyKey = `test-reserve-${Date.now()}-${Math.random().toString(36).slice(2)}`
    const res = await this.client.post('/bookings/reserve', data, {
      headers: { 'X-Idempotency-Key': idempotencyKey },
    })
    return res.data
  }

  async getBooking(bookingId: string) {
    const res = await this.client.get(`/bookings/${bookingId}`)
    return res.data
  }

  async getUserBookings() {
    const res = await this.client.get('/bookings')
    return res.data
  }

  async releaseBooking(bookingId: string) {
    const res = await this.client.post(`/bookings/${bookingId}/release`)
    return res.data
  }

  // Payment endpoints
  async createPaymentIntent(data: { booking_id: string; amount: number }) {
    const idempotencyKey = `test-payment-${Date.now()}-${Math.random().toString(36).slice(2)}`
    const res = await this.client.post('/payments/create-intent', data, {
      headers: { 'X-Idempotency-Key': idempotencyKey },
    })
    return res.data
  }

  // Organizer endpoints
  async getMyEvents(params?: { limit?: number; offset?: number }) {
    const res = await this.client.get('/events/my', { params })
    return res.data
  }

  async createEvent(data: {
    name: string
    description: string
    short_description: string
    venue_name: string
    venue_address: string
    city: string
    country: string
    max_tickets_per_user: number
  }) {
    const res = await this.client.post('/events', data)
    return res.data
  }

  async updateEvent(eventId: string, data: Partial<{
    name: string
    description: string
    status: string
  }>) {
    const res = await this.client.put(`/events/${eventId}`, data)
    return res.data
  }

  async createShow(eventId: string, data: {
    name: string
    show_date: string
    start_time: string
    end_time: string
  }) {
    const res = await this.client.post(`/events/${eventId}/shows`, data)
    return res.data
  }

  async createZone(showId: string, data: {
    name: string
    price: number
    total_seats: number
    min_per_order: number
    max_per_order: number
  }) {
    const res = await this.client.post(`/shows/${showId}/zones`, data)
    return res.data
  }

  // Health check
  async healthCheck() {
    const res = await this.client.get('/health')
    return res.data
  }
}

/**
 * Generate random email for testing
 */
export function randomEmail(): string {
  return `test-${Date.now()}-${Math.random().toString(36).slice(2)}@test.com`
}

/**
 * Generate random string
 */
export function randomString(length: number = 8): string {
  return Math.random().toString(36).slice(2, 2 + length)
}
