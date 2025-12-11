"use client"

import { useState, useEffect, useCallback } from "react"
import { Card } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Separator } from "@/components/ui/separator"
import { CreditCard, Smartphone, Clock, Shield, Lock, Calendar, MapPin, Ticket, AlertTriangle } from "lucide-react"
import { useRouter } from "next/navigation"
import { bookingApi, paymentApi } from "@/lib/api/booking"
import { eventsApi } from "@/lib/api/events"
import type { EventResponse, ShowResponse, ShowZoneResponse, ReserveSeatsResponse } from "@/lib/api/types"
import { ApiRequestError } from "@/lib/api/client"

type CheckoutState = "loading" | "reserving" | "ready" | "processing" | "success" | "error" | "timeout"

interface QueueData {
  eventId: string
  showId: string
  tickets: Record<string, number>
  total: number
  queuePass: string
  queuePassExpiresAt: string
}

export default function CheckoutPage() {
  const router = useRouter()

  // Queue data from sessionStorage
  const [queueData, setQueueData] = useState<QueueData | null>(null)

  // Event/Show/Zone data
  const [event, setEvent] = useState<EventResponse | null>(null)
  const [show, setShow] = useState<ShowResponse | null>(null)
  const [zones, setZones] = useState<ShowZoneResponse[]>([])

  // Booking state
  const [checkoutState, setCheckoutState] = useState<CheckoutState>("loading")
  const [reservation, setReservation] = useState<ReserveSeatsResponse | null>(null)
  const [error, setError] = useState<string>("")

  // Timer state
  const [timeLeft, setTimeLeft] = useState(600) // 10 minutes default

  // Payment form
  const [paymentMethod, setPaymentMethod] = useState("card")
  const [cardNumber, setCardNumber] = useState("")
  const [cardExpiry, setCardExpiry] = useState("")
  const [cardCvv, setCardCvv] = useState("")
  const [cardName, setCardName] = useState("")
  const [isSubmitting, setIsSubmitting] = useState(false)

  // Load queue data from sessionStorage
  useEffect(() => {
    if (typeof window === "undefined") return

    const eventId = sessionStorage.getItem("queue_event_id")
    const showId = sessionStorage.getItem("queue_show_id")
    const ticketsStr = sessionStorage.getItem("queue_tickets")
    const totalStr = sessionStorage.getItem("queue_total")
    const queuePass = sessionStorage.getItem("queue_pass")
    const queuePassExpiresAt = sessionStorage.getItem("queue_pass_expires_at")

    if (!eventId || !queuePass) {
      setError("No queue pass found. Please join the queue first.")
      setCheckoutState("error")
      return
    }

    const tickets = ticketsStr ? JSON.parse(ticketsStr) : {}
    const total = totalStr ? parseInt(totalStr, 10) : 0

    setQueueData({
      eventId,
      showId: showId || "",
      tickets,
      total,
      queuePass,
      queuePassExpiresAt: queuePassExpiresAt || "",
    })

    // Calculate time left from queue pass expiry
    if (queuePassExpiresAt) {
      const expiresAt = new Date(queuePassExpiresAt).getTime()
      const now = Date.now()
      const remaining = Math.max(0, Math.floor((expiresAt - now) / 1000))
      setTimeLeft(remaining)
    }
  }, [])

  // Fetch event details
  useEffect(() => {
    if (!queueData?.eventId) return

    const fetchEventDetails = async () => {
      try {
        const eventData = await eventsApi.getEvent(queueData.eventId)
        setEvent(eventData)

        if (queueData.showId && eventData.slug) {
          // Use slug from fetched event to get shows
          const shows = await eventsApi.getEventShowsBySlug(eventData.slug)
          const showData = shows.find((s: ShowResponse) => s.id === queueData.showId)
          if (showData) {
            setShow(showData)
            const zonesData = await eventsApi.getShowZones(showData.id)
            setZones(zonesData)
          }
        }

        setCheckoutState("reserving")
      } catch (err) {
        console.error("Failed to fetch event details:", err)
        setError("Failed to load event details")
        setCheckoutState("error")
      }
    }

    fetchEventDetails()
  }, [queueData])

  // Reserve seats when ready
  useEffect(() => {
    if (checkoutState !== "reserving" || !queueData || !event) return

    const reserveSeats = async () => {
      try {
        // Get first zone from tickets (simplified - in real app may have multiple zones)
        const zoneEntries = Object.entries(queueData.tickets)
        if (zoneEntries.length === 0) {
          setError("No tickets selected")
          setCheckoutState("error")
          return
        }

        const [zoneId, quantity] = zoneEntries[0]
        const zone = zones.find(z => z.id === zoneId)

        const reservationData = await bookingApi.reserveSeats(
          {
            event_id: queueData.eventId,
            zone_id: zoneId,
            show_id: queueData.showId || undefined,
            quantity: quantity as number,
            unit_price: zone?.price,
          },
          queueData.queuePass
        )

        setReservation(reservationData)

        // Update timer based on reservation expiry
        if (reservationData.expires_at) {
          const expiresAt = new Date(reservationData.expires_at).getTime()
          const now = Date.now()
          const remaining = Math.max(0, Math.floor((expiresAt - now) / 1000))
          setTimeLeft(remaining)
        }

        setCheckoutState("ready")
      } catch (err) {
        console.error("Failed to reserve seats:", err)
        if (err instanceof ApiRequestError) {
          if (err.code === "QUEUE_PASS_EXPIRED" || err.code === "INVALID_QUEUE_PASS") {
            setError("Your queue pass has expired. Please rejoin the queue.")
          } else if (err.code === "SEATS_NOT_AVAILABLE") {
            setError("Sorry, the requested seats are no longer available.")
          } else {
            setError(err.message)
          }
        } else {
          setError("Failed to reserve seats")
        }
        setCheckoutState("error")
      }
    }

    reserveSeats()
  }, [checkoutState, queueData, event, zones])

  // Countdown timer
  useEffect(() => {
    if (checkoutState !== "ready" || timeLeft <= 0) return

    const timer = setInterval(() => {
      setTimeLeft((prev) => {
        if (prev <= 1) {
          clearInterval(timer)
          setCheckoutState("timeout")
          return 0
        }
        return prev - 1
      })
    }, 1000)

    return () => clearInterval(timer)
  }, [checkoutState, timeLeft])

  // Handle timeout - release reservation and redirect
  useEffect(() => {
    if (checkoutState !== "timeout") return

    const handleTimeout = async () => {
      if (reservation?.booking_id) {
        try {
          await bookingApi.releaseBooking(reservation.booking_id)
        } catch (err) {
          console.error("Failed to release booking:", err)
        }
      }

      // Clear session storage
      clearQueueSession()

      // Redirect after short delay
      setTimeout(() => {
        router.push("/")
      }, 3000)
    }

    handleTimeout()
  }, [checkoutState, reservation, router])

  const clearQueueSession = () => {
    if (typeof window === "undefined") return
    sessionStorage.removeItem("queue_token")
    sessionStorage.removeItem("queue_event_id")
    sessionStorage.removeItem("queue_show_id")
    sessionStorage.removeItem("queue_tickets")
    sessionStorage.removeItem("queue_total")
    sessionStorage.removeItem("queue_pass")
    sessionStorage.removeItem("queue_pass_expires_at")
  }

  const formatTime = (seconds: number) => {
    const mins = Math.floor(seconds / 60)
    const secs = seconds % 60
    return `${mins.toString().padStart(2, "0")}:${secs.toString().padStart(2, "0")}`
  }

  const isUrgent = timeLeft < 120 // Less than 2 minutes

  // Calculate totals
  const getOrderSummary = useCallback(() => {
    if (!queueData || !zones.length) {
      return { items: [], subtotal: 0, serviceFee: 0, total: 0 }
    }

    const items = Object.entries(queueData.tickets).map(([zoneId, quantity]) => {
      const zone = zones.find(z => z.id === zoneId)
      return {
        zoneId,
        zoneName: zone?.name || "Unknown Zone",
        quantity: quantity as number,
        price: zone?.price || 0,
        subtotal: (zone?.price || 0) * (quantity as number),
      }
    })

    const subtotal = items.reduce((sum, item) => sum + item.subtotal, 0)
    const serviceFee = Math.round(subtotal * 0.05) // 5% service fee
    const total = subtotal + serviceFee

    return { items, subtotal, serviceFee, total }
  }, [queueData, zones])

  const orderSummary = getOrderSummary()

  // Handle payment submission
  const handlePayment = async () => {
    if (!reservation?.booking_id) return

    setIsSubmitting(true)

    try {
      // Create payment (mock)
      const payment = await paymentApi.createPayment({
        booking_id: reservation.booking_id,
        payment_method: paymentMethod,
        amount: orderSummary.total,
      })

      // Confirm booking with payment
      await bookingApi.confirmBooking(reservation.booking_id, {
        payment_id: payment.id,
      })

      setCheckoutState("success")

      // Clear session and redirect to confirmation
      clearQueueSession()

      setTimeout(() => {
        router.push(`/booking/confirmation?booking_id=${reservation.booking_id}`)
      }, 2000)
    } catch (err) {
      console.error("Payment failed:", err)
      if (err instanceof ApiRequestError) {
        setError(err.message)
      } else {
        setError("Payment failed. Please try again.")
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  // Handle cancel
  const handleCancel = async () => {
    if (reservation?.booking_id) {
      try {
        await bookingApi.releaseBooking(reservation.booking_id)
      } catch (err) {
        console.error("Failed to release booking:", err)
      }
    }

    clearQueueSession()
    router.push("/")
  }

  // Loading state
  if (checkoutState === "loading" || checkoutState === "reserving") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center">
        <div className="text-center space-y-4">
          <div className="w-16 h-16 border-4 border-[#d4af37]/30 border-t-[#d4af37] rounded-full animate-spin mx-auto" />
          <p className="text-gray-400">
            {checkoutState === "loading" ? "Loading checkout..." : "Reserving your seats..."}
          </p>
        </div>
      </div>
    )
  }

  // Error state
  if (checkoutState === "error") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center p-4">
        <Card className="bg-[#1a1a1a] border-red-800 p-8 max-w-md text-center">
          <div className="w-16 h-16 rounded-full border-2 border-red-500 flex items-center justify-center mx-auto mb-4">
            <AlertTriangle className="w-8 h-8 text-red-500" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Checkout Failed</h1>
          <p className="text-gray-400 mb-6">{error}</p>
          <Button onClick={() => router.push("/")} className="bg-[#d4af37] hover:bg-[#d4af37]/90 text-black">
            Back to Events
          </Button>
        </Card>
      </div>
    )
  }

  // Timeout state
  if (checkoutState === "timeout") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center p-4">
        <Card className="bg-[#1a1a1a] border-yellow-800 p-8 max-w-md text-center">
          <div className="w-16 h-16 rounded-full border-2 border-yellow-500 flex items-center justify-center mx-auto mb-4">
            <Clock className="w-8 h-8 text-yellow-500" />
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Session Expired</h1>
          <p className="text-gray-400 mb-6">
            Your reservation has timed out. Your seats have been released. Redirecting...
          </p>
        </Card>
      </div>
    )
  }

  // Success state
  if (checkoutState === "success") {
    return (
      <div className="min-h-screen bg-[#0a0a0a] flex items-center justify-center p-4">
        <Card className="bg-[#1a1a1a] border-green-800 p-8 max-w-md text-center">
          <div className="w-16 h-16 rounded-full border-2 border-green-500 flex items-center justify-center mx-auto mb-4">
            <svg className="w-8 h-8 text-green-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
          </div>
          <h1 className="text-2xl font-bold text-white mb-2">Payment Successful!</h1>
          <p className="text-gray-400 mb-6">Redirecting to your booking confirmation...</p>
        </Card>
      </div>
    )
  }

  // Ready state - main checkout form
  return (
    <div className="min-h-screen bg-[#0a0a0a]">
      <div className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-3xl font-bold text-white">Checkout</h1>
          <p className="mt-2 text-gray-400">Complete your booking securely</p>
        </div>

        {/* Two Column Layout */}
        <div className="grid gap-8 lg:grid-cols-2">
          {/* Left Column - Order Summary */}
          <div className="lg:order-1">
            <Card className="overflow-hidden border-0 bg-[#141414]">
              <div className="p-6">
                <h2 className="mb-6 text-xl font-semibold text-white">Order Summary</h2>

                {/* Event Image */}
                {event?.banner_url && (
                  <div className="mb-6 overflow-hidden rounded-lg">
                    <img src={event.banner_url} alt={event.name} className="h-48 w-full object-cover" />
                  </div>
                )}

                {/* Event Details */}
                <div className="space-y-4">
                  <div>
                    <h3 className="text-lg font-semibold text-white">{event?.name || "Event"}</h3>
                  </div>

                  <div className="flex items-start gap-3 text-sm text-gray-300">
                    <Calendar className="mt-0.5 h-4 w-4 shrink-0 text-[#d4af37]" />
                    <div>
                      <div>
                        {show?.show_date
                          ? new Date(show.show_date).toLocaleDateString("en-US", {
                              weekday: "long",
                              month: "long",
                              day: "numeric",
                              year: "numeric",
                            })
                          : "Date TBA"}
                      </div>
                      <div className="text-gray-400">
                        {show?.start_time
                          ? new Date(show.start_time).toLocaleTimeString("en-US", {
                              hour: "numeric",
                              minute: "2-digit",
                              hour12: true,
                            })
                          : "Time TBA"}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-start gap-3 text-sm text-gray-300">
                    <MapPin className="mt-0.5 h-4 w-4 shrink-0 text-[#d4af37]" />
                    <div>{event?.venue_name || "Venue TBA"}</div>
                  </div>

                  <div className="flex items-start gap-3 text-sm text-gray-300">
                    <Ticket className="mt-0.5 h-4 w-4 shrink-0 text-[#d4af37]" />
                    <div>
                      {orderSummary.items.map((item) => (
                        <div key={item.zoneId}>
                          <div>{item.zoneName}</div>
                          <div className="text-gray-400">Quantity: {item.quantity}</div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>

                <Separator className="my-6 bg-gray-700" />

                {/* Price Breakdown */}
                <div className="space-y-3">
                  {orderSummary.items.map((item) => (
                    <div key={item.zoneId} className="flex justify-between text-sm text-gray-300">
                      <span>
                        {item.zoneName} x {item.quantity}
                      </span>
                      <span>฿{item.subtotal.toLocaleString()}</span>
                    </div>
                  ))}
                  <div className="flex justify-between text-sm text-gray-300">
                    <span>Service Fee (5%)</span>
                    <span>฿{orderSummary.serviceFee.toLocaleString()}</span>
                  </div>

                  <Separator className="bg-gray-700" />

                  <div className="flex justify-between text-lg font-bold text-white">
                    <span>Total</span>
                    <span className="text-[#d4af37]">฿{orderSummary.total.toLocaleString()}</span>
                  </div>
                </div>
              </div>
            </Card>
          </div>

          {/* Right Column - Payment Section */}
          <div className="lg:order-2">
            <Card className="border-0 bg-[#141414]">
              <div className="p-6">
                {/* Countdown Timer */}
                <div
                  className={`mb-6 flex items-center justify-center gap-2 rounded-lg p-4 ${
                    isUrgent ? "bg-red-950/50" : "bg-gray-800/50"
                  }`}
                >
                  <Clock className={`h-5 w-5 ${isUrgent ? "text-red-400" : "text-gray-400"}`} />
                  <span className={`font-mono text-lg font-semibold ${isUrgent ? "text-red-400" : "text-gray-300"}`}>
                    Complete in {formatTime(timeLeft)}
                  </span>
                </div>

                <h2 className="mb-6 text-xl font-semibold text-white">Payment Details</h2>

                {/* Payment Method Selector */}
                <div className="mb-6">
                  <Label className="mb-3 block text-sm font-medium text-gray-300">Payment Method</Label>
                  <RadioGroup value={paymentMethod} onValueChange={setPaymentMethod}>
                    <div
                      className={`flex items-center space-x-3 rounded-lg border p-4 cursor-pointer ${
                        paymentMethod === "card" ? "border-[#d4af37] bg-[#1a1a1a]" : "border-[#2a2a2a]"
                      }`}
                      onClick={() => setPaymentMethod("card")}
                    >
                      <RadioGroupItem value="card" id="card" />
                      <CreditCard className="h-5 w-5 text-gray-400" />
                      <Label htmlFor="card" className="flex-1 cursor-pointer text-white">
                        Credit / Debit Card
                      </Label>
                    </div>

                    <div
                      className={`mt-3 flex items-center space-x-3 rounded-lg border p-4 cursor-pointer ${
                        paymentMethod === "promptpay" ? "border-[#d4af37] bg-[#1a1a1a]" : "border-[#2a2a2a]"
                      }`}
                      onClick={() => setPaymentMethod("promptpay")}
                    >
                      <RadioGroupItem value="promptpay" id="promptpay" />
                      <Smartphone className="h-5 w-5 text-gray-400" />
                      <Label htmlFor="promptpay" className="flex-1 cursor-pointer text-white">
                        PromptPay
                      </Label>
                    </div>
                  </RadioGroup>
                </div>

                {/* Payment Form */}
                {paymentMethod === "card" && (
                  <div className="space-y-4">
                    <div>
                      <Label htmlFor="cardNumber" className="text-gray-300">
                        Card Number
                      </Label>
                      <Input
                        id="cardNumber"
                        placeholder="1234 5678 9012 3456"
                        value={cardNumber}
                        onChange={(e) => setCardNumber(e.target.value)}
                        className="mt-1.5 border-gray-700 bg-black/30 text-white placeholder:text-gray-500"
                        maxLength={19}
                      />
                    </div>

                    <div className="grid grid-cols-2 gap-4">
                      <div>
                        <Label htmlFor="expiry" className="text-gray-300">
                          Expiry Date
                        </Label>
                        <Input
                          id="expiry"
                          placeholder="MM / YY"
                          value={cardExpiry}
                          onChange={(e) => setCardExpiry(e.target.value)}
                          className="mt-1.5 border-gray-700 bg-black/30 text-white placeholder:text-gray-500"
                          maxLength={7}
                        />
                      </div>
                      <div>
                        <Label htmlFor="cvv" className="text-gray-300">
                          CVV
                        </Label>
                        <Input
                          id="cvv"
                          placeholder="123"
                          type="password"
                          value={cardCvv}
                          onChange={(e) => setCardCvv(e.target.value)}
                          className="mt-1.5 border-gray-700 bg-black/30 text-white placeholder:text-gray-500"
                          maxLength={4}
                        />
                      </div>
                    </div>

                    <div>
                      <Label htmlFor="cardName" className="text-gray-300">
                        Cardholder Name
                      </Label>
                      <Input
                        id="cardName"
                        placeholder="JOHN DOE"
                        value={cardName}
                        onChange={(e) => setCardName(e.target.value)}
                        className="mt-1.5 border-gray-700 bg-black/30 text-white placeholder:text-gray-500"
                      />
                    </div>
                  </div>
                )}

                {paymentMethod === "promptpay" && (
                  <div className="rounded-lg bg-gray-800/50 p-6 text-center">
                    <Smartphone className="mx-auto mb-3 h-12 w-12 text-gray-400" />
                    <p className="text-sm text-gray-300">
                      You will receive a PromptPay QR code after clicking the pay button
                    </p>
                  </div>
                )}

                {/* Error message */}
                {error && (
                  <div className="mt-4 p-3 bg-red-950/50 border border-red-800 rounded-lg">
                    <p className="text-sm text-red-400">{error}</p>
                  </div>
                )}

                {/* Pay Button */}
                <Button
                  onClick={handlePayment}
                  disabled={isSubmitting || timeLeft <= 0}
                  className="mt-6 w-full py-6 text-lg font-semibold bg-[#d4af37] hover:bg-[#d4af37]/90 text-[#0a0a0a] disabled:opacity-50"
                >
                  {isSubmitting ? "Processing..." : `Pay ฿${orderSummary.total.toLocaleString()}`}
                </Button>

                {/* Cancel Button */}
                <Button
                  variant="ghost"
                  onClick={handleCancel}
                  className="mt-3 w-full text-gray-400 hover:text-gray-300"
                >
                  Cancel and Release Seats
                </Button>

                {/* Trust Badges */}
                <div className="mt-6 flex flex-wrap items-center justify-center gap-4 border-t border-gray-700 pt-6">
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <Shield className="h-4 w-4 text-[#d4af37]" />
                    <span>SSL Encrypted</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <Lock className="h-4 w-4 text-[#d4af37]" />
                    <span>Secure Payment</span>
                  </div>
                  <div className="flex items-center gap-2 text-sm text-gray-400">
                    <CreditCard className="h-4 w-4 text-[#d4af37]" />
                    <span>PCI Compliant</span>
                  </div>
                </div>

                <p className="mt-4 text-center text-xs text-gray-500">
                  Your payment information is processed securely. We do not store credit card details.
                </p>
              </div>
            </Card>
          </div>
        </div>
      </div>
    </div>
  )
}
