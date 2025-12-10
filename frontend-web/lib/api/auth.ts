import { apiClient } from "./client"
import type {
  LoginRequest,
  RegisterRequest,
  RefreshTokenRequest,
  AuthResponse,
  UserResponse,
} from "./types"

export const authApi = {
  async login(data: LoginRequest): Promise<AuthResponse> {
    const response = await apiClient.post<AuthResponse>("/auth/login", data)
    apiClient.setAccessToken(response.access_token)
    if (typeof window !== "undefined") {
      localStorage.setItem("refresh_token", response.refresh_token)
      localStorage.setItem("user", JSON.stringify(response.user))
    }
    return response
  },

  async register(data: RegisterRequest): Promise<AuthResponse> {
    const response = await apiClient.post<AuthResponse>("/auth/register", data)
    apiClient.setAccessToken(response.access_token)
    if (typeof window !== "undefined") {
      localStorage.setItem("refresh_token", response.refresh_token)
      localStorage.setItem("user", JSON.stringify(response.user))
    }
    return response
  },

  async refreshToken(): Promise<AuthResponse> {
    const refreshToken = typeof window !== "undefined"
      ? localStorage.getItem("refresh_token")
      : null

    if (!refreshToken) {
      throw new Error("No refresh token available")
    }

    const data: RefreshTokenRequest = { refresh_token: refreshToken }
    const response = await apiClient.post<AuthResponse>("/auth/refresh", data)
    apiClient.setAccessToken(response.access_token)
    if (typeof window !== "undefined") {
      localStorage.setItem("refresh_token", response.refresh_token)
      localStorage.setItem("user", JSON.stringify(response.user))
    }
    return response
  },

  logout() {
    apiClient.clearTokens()
  },

  getStoredUser(): UserResponse | null {
    if (typeof window === "undefined") return null
    const user = localStorage.getItem("user")
    if (!user || user === "undefined" || user === "null") return null
    try {
      return JSON.parse(user)
    } catch {
      return null
    }
  },

  isAuthenticated(): boolean {
    return !!apiClient.getAccessToken()
  },
}
