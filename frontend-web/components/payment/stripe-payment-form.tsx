"use client"

import { useState, useEffect } from "react"
import {
  Elements,
  PaymentElement,
  useStripe,
  useElements,
} from "@stripe/react-stripe-js"
import { Stripe, StripeElementsOptions } from "@stripe/stripe-js"
import { Button } from "@/components/ui/button"
import { Loader2, Lock, AlertCircle } from "lucide-react"
import type { PaymentMethod } from "@/lib/api/payment"

interface StripePaymentFormProps {
  clientSecret: string
  stripe: Stripe | null
  amount: number
  onSuccess: (paymentIntentId: string) => void
  onError: (error: string) => void
  disabled?: boolean
  savedPaymentMethods?: PaymentMethod[]
  selectedPaymentMethod?: string | null
}

interface CheckoutFormProps {
  amount: number
  clientSecret: string
  onSuccess: (paymentIntentId: string) => void
  onError: (error: string) => void
  disabled?: boolean
  selectedPaymentMethod?: string | null
}

function CheckoutForm({
  amount,
  clientSecret,
  onSuccess,
  onError,
  disabled,
  selectedPaymentMethod,
}: CheckoutFormProps) {
  const stripe = useStripe()
  const elements = useElements()
  const [isLoading, setIsLoading] = useState(false)
  const [errorMessage, setErrorMessage] = useState<string | null>(null)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()

    if (!stripe) {
      return
    }

    setIsLoading(true)
    setErrorMessage(null)

    try {
      // If using saved payment method
      if (selectedPaymentMethod) {
        const { error, paymentIntent } = await stripe.confirmCardPayment(
          clientSecret,
          {
            payment_method: selectedPaymentMethod,
            return_url: `${window.location.origin}/booking/confirmation`,
          }
        )

        if (error) {
          setErrorMessage(error.message || "Payment failed")
          onError(error.message || "Payment failed")
        } else if (paymentIntent && paymentIntent.status === "succeeded") {
          onSuccess(paymentIntent.id)
        } else if (paymentIntent && paymentIntent.status === "requires_action") {
          setErrorMessage("Additional authentication required")
        } else {
          setErrorMessage("Payment was not completed")
          onError("Payment was not completed")
        }
      } else {
        // Using new card via PaymentElement
        if (!elements) {
          setErrorMessage("Payment form not ready")
          return
        }

        const { error, paymentIntent } = await stripe.confirmPayment({
          elements,
          confirmParams: {
            return_url: `${window.location.origin}/booking/confirmation`,
          },
          redirect: "if_required",
        })

        if (error) {
          setErrorMessage(error.message || "Payment failed")
          onError(error.message || "Payment failed")
        } else if (paymentIntent && paymentIntent.status === "succeeded") {
          onSuccess(paymentIntent.id)
        } else if (paymentIntent && paymentIntent.status === "requires_action") {
          setErrorMessage("Additional authentication required")
        } else {
          setErrorMessage("Payment was not completed")
          onError("Payment was not completed")
        }
      }
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : "Payment failed"
      setErrorMessage(errorMsg)
      onError(errorMsg)
    } finally {
      setIsLoading(false)
    }
  }

  // Show PaymentElement only when using new card
  const showPaymentElement = selectedPaymentMethod === null || selectedPaymentMethod === undefined

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {showPaymentElement && (
        <div className="rounded-lg border border-gray-700 bg-black/30 p-4">
          <PaymentElement
            options={{
              layout: "tabs",
            }}
          />
        </div>
      )}

      {errorMessage && (
        <div className="flex items-center gap-2 rounded-lg border border-red-800 bg-red-950/50 p-3 text-sm text-red-400">
          <AlertCircle className="h-4 w-4 shrink-0" />
          <span>{errorMessage}</span>
        </div>
      )}

      <Button
        type="submit"
        disabled={!stripe || (showPaymentElement && !elements) || isLoading || disabled}
        className="w-full py-6 text-lg font-semibold bg-[#d4af37] hover:bg-[#d4af37]/90 text-[#0a0a0a] disabled:opacity-50"
      >
        {isLoading ? (
          <>
            <Loader2 className="mr-2 h-5 w-5 animate-spin" />
            Processing...
          </>
        ) : (
          <>
            <Lock className="mr-2 h-5 w-5" />
            Pay ฿{amount.toLocaleString()}
          </>
        )}
      </Button>

      <p className="text-center text-xs text-gray-500">
        Your payment is processed securely by Stripe. We never store your card details.
      </p>
    </form>
  )
}

export function StripePaymentForm({
  clientSecret,
  stripe,
  amount,
  onSuccess,
  onError,
  disabled,
  selectedPaymentMethod,
}: StripePaymentFormProps) {
  const [isReady, setIsReady] = useState(false)

  useEffect(() => {
    if (stripe && clientSecret) {
      setIsReady(true)
    }
  }, [stripe, clientSecret])

  if (!isReady) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-8 w-8 animate-spin text-[#d4af37]" />
        <span className="ml-3 text-gray-400">Loading payment form...</span>
      </div>
    )
  }

  const options: StripeElementsOptions = {
    clientSecret,
    appearance: {
      theme: "night",
      variables: {
        colorPrimary: "#d4af37",
        colorBackground: "#1a1a1a",
        colorText: "#ffffff",
        colorTextSecondary: "#9ca3af",
        colorDanger: "#ef4444",
        fontFamily: "system-ui, sans-serif",
        borderRadius: "8px",
        spacingUnit: "4px",
      },
      rules: {
        ".Input": {
          backgroundColor: "rgba(0, 0, 0, 0.3)",
          border: "1px solid #374151",
        },
        ".Input:focus": {
          border: "1px solid #d4af37",
          boxShadow: "0 0 0 1px #d4af37",
        },
        ".Label": {
          color: "#9ca3af",
        },
        ".Tab": {
          backgroundColor: "#1a1a1a",
          border: "1px solid #374151",
        },
        ".Tab--selected": {
          backgroundColor: "#2a2a2a",
          border: "1px solid #d4af37",
        },
        ".TabIcon": {
          color: "#9ca3af",
        },
        ".TabIcon--selected": {
          color: "#d4af37",
        },
      },
    },
  }

  return (
    <Elements stripe={stripe} options={options}>
      <CheckoutForm
        amount={amount}
        clientSecret={clientSecret}
        onSuccess={onSuccess}
        onError={onError}
        disabled={disabled}
        selectedPaymentMethod={selectedPaymentMethod}
      />
    </Elements>
  )
}
