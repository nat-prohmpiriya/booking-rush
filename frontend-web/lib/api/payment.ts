import { apiClient } from "./client"

export interface PortalSessionResponse {
  url: string
}

export interface PaymentMethod {
  id: string
  type: string
  brand: string
  last4: string
  exp_month: number
  exp_year: number
  is_default: boolean
}

export interface PaymentMethodsResponse {
  payment_methods: PaymentMethod[]
  total: number
}

export const paymentApi = {
  /**
   * Create a Stripe Customer Portal session
   * Returns a URL to redirect the user to for managing payment methods
   */
  createPortalSession: async (returnUrl: string): Promise<PortalSessionResponse> => {
    return apiClient.post<PortalSessionResponse>("/payments/portal", {
      return_url: returnUrl,
    })
  },

  /**
   * List saved payment methods for the current user
   */
  listPaymentMethods: async (): Promise<PaymentMethodsResponse> => {
    return apiClient.get<PaymentMethodsResponse>("/payments/methods")
  },
}
