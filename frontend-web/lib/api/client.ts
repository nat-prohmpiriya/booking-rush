import type { ApiError } from "./types"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1"

type HttpMethod = "GET" | "POST" | "PUT" | "PATCH" | "DELETE"

interface RequestOptions {
  method?: HttpMethod
  body?: unknown
  headers?: Record<string, string>
  requireAuth?: boolean
}

class ApiClient {
  private baseUrl: string
  private accessToken: string | null = null

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl
    if (typeof window !== "undefined") {
      this.accessToken = localStorage.getItem("access_token")
    }
  }

  setAccessToken(token: string | null) {
    this.accessToken = token
    if (typeof window !== "undefined") {
      if (token) {
        localStorage.setItem("access_token", token)
      } else {
        localStorage.removeItem("access_token")
      }
    }
  }

  getAccessToken(): string | null {
    if (typeof window !== "undefined" && !this.accessToken) {
      this.accessToken = localStorage.getItem("access_token")
    }
    return this.accessToken
  }

  clearTokens() {
    this.accessToken = null
    if (typeof window !== "undefined") {
      localStorage.removeItem("access_token")
      localStorage.removeItem("refresh_token")
      localStorage.removeItem("user")
    }
  }

  private async request<T>(endpoint: string, options: RequestOptions = {}): Promise<T> {
    const { method = "GET", body, headers = {}, requireAuth = false } = options

    const requestHeaders: Record<string, string> = {
      "Content-Type": "application/json",
      ...headers,
    }

    if (requireAuth || this.accessToken) {
      const token = this.getAccessToken()
      if (token) {
        requestHeaders["Authorization"] = `Bearer ${token}`
      } else if (requireAuth) {
        throw new Error("Authentication required")
      }
    }

    const config: RequestInit = {
      method,
      headers: requestHeaders,
      credentials: "include",
    }

    if (body && method !== "GET") {
      config.body = JSON.stringify(body)
    }

    const url = `${this.baseUrl}${endpoint}`
    const response = await fetch(url, config)

    if (!response.ok) {
      let errorData: ApiError
      try {
        errorData = await response.json()
      } catch {
        errorData = {
          error: response.statusText,
          message: `Request failed with status ${response.status}`,
        }
      }

      if (response.status === 401) {
        this.clearTokens()
        if (typeof window !== "undefined") {
          window.dispatchEvent(new CustomEvent("auth:unauthorized"))
        }
      }

      throw new ApiRequestError(
        errorData.message || errorData.error,
        response.status,
        errorData.code
      )
    }

    if (response.status === 204) {
      return {} as T
    }

    const json = await response.json()
    // Backend wraps response in { success: boolean, data: T }
    // For paginated responses: { success: boolean, data: T[], meta: {...} }
    if (json && typeof json === "object" && "success" in json) {
      // Paginated response with meta
      if ("meta" in json) {
        return { data: json.data, meta: json.meta } as T
      }
      // Regular response with data wrapper
      if ("data" in json) {
        return json.data as T
      }
    }
    return json as T
  }

  async get<T>(endpoint: string, options?: Omit<RequestOptions, "method" | "body">): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: "GET" })
  }

  async post<T>(endpoint: string, body?: unknown, options?: Omit<RequestOptions, "method" | "body">): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: "POST", body })
  }

  async put<T>(endpoint: string, body?: unknown, options?: Omit<RequestOptions, "method" | "body">): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: "PUT", body })
  }

  async patch<T>(endpoint: string, body?: unknown, options?: Omit<RequestOptions, "method" | "body">): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: "PATCH", body })
  }

  async delete<T>(endpoint: string, options?: Omit<RequestOptions, "method" | "body">): Promise<T> {
    return this.request<T>(endpoint, { ...options, method: "DELETE" })
  }
}

export class ApiRequestError extends Error {
  status: number
  code?: string

  constructor(message: string, status: number, code?: string) {
    super(message)
    this.name = "ApiRequestError"
    this.status = status
    this.code = code
  }
}

export const apiClient = new ApiClient(API_BASE_URL)
