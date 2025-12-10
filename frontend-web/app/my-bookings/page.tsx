"use client"

import { useState, useEffect } from "react"
import { useRouter } from "next/navigation"
import Link from "next/link"
import { Header } from "@/components/header"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { useAuth } from "@/contexts/auth-context"
import { bookingApi } from "@/lib/api/booking"
import type { BookingResponse } from "@/lib/api/types"
import {
  Ticket,
  Calendar,
  Clock,
  MapPin,
  ChevronRight,
  Filter,
  Search,
  QrCode,
  Download,
  MoreHorizontal,
  AlertCircle,
  CheckCircle2,
  XCircle,
  Timer,
} from "lucide-react"
import { Input } from "@/components/ui/input"

// Mock data for bookings (since API might not have full event info)
interface BookingWithEvent extends BookingResponse {
  event?: {
    id: string
    title: string
    venue: string
    date: string
    time: string
    image: string
  }
  tickets?: {
    zone: string
    quantity: number
    price: number
  }
}

const MOCK_BOOKINGS: BookingWithEvent[] = [
  {
    id: "booking-001",
    user_id: "user-1",
    reservation_id: "res-001",
    status: "confirmed",
    total_amount: 4500,
    created_at: "2025-12-15T10:30:00Z",
    updated_at: "2025-12-15T10:30:00Z",
    event: {
      id: "event-1",
      title: "BLACKPINK World Tour 2025",
      venue: "Rajamangala National Stadium, Bangkok",
      date: "Jan 15, 2025",
      time: "7:00 PM",
      image: "/images/events/event-1.jpg",
    },
    tickets: {
      zone: "VIP Standing",
      quantity: 2,
      price: 2250,
    },
  },
  {
    id: "booking-002",
    user_id: "user-1",
    reservation_id: "res-002",
    status: "pending",
    total_amount: 3200,
    created_at: "2025-12-10T14:20:00Z",
    updated_at: "2025-12-10T14:20:00Z",
    event: {
      id: "event-2",
      title: "Ed Sheeran + - = ÷ x Tour",
      venue: "Impact Arena, Bangkok",
      date: "Feb 20, 2025",
      time: "8:00 PM",
      image: "/images/events/event-2.jpg",
    },
    tickets: {
      zone: "Gold Section",
      quantity: 2,
      price: 1600,
    },
  },
  {
    id: "booking-003",
    user_id: "user-1",
    reservation_id: "res-003",
    status: "completed",
    total_amount: 1800,
    created_at: "2025-11-20T09:15:00Z",
    updated_at: "2025-11-20T09:15:00Z",
    event: {
      id: "event-3",
      title: "Jazz Festival 2025",
      venue: "Lumpini Park, Bangkok",
      date: "Nov 25, 2025",
      time: "6:00 PM",
      image: "/images/events/event-3.jpg",
    },
    tickets: {
      zone: "General Admission",
      quantity: 3,
      price: 600,
    },
  },
  {
    id: "booking-004",
    user_id: "user-1",
    reservation_id: "res-004",
    status: "cancelled",
    total_amount: 5500,
    created_at: "2025-10-05T16:45:00Z",
    updated_at: "2025-10-06T08:30:00Z",
    event: {
      id: "event-4",
      title: "Taylor Swift Eras Tour",
      venue: "Rajamangala National Stadium, Bangkok",
      date: "Oct 20, 2025",
      time: "7:30 PM",
      image: "/images/events/event-4.jpg",
    },
    tickets: {
      zone: "Premium Seat",
      quantity: 1,
      price: 5500,
    },
  },
]

function BookingCardSkeleton() {
  return (
    <div className="glass rounded-xl p-4 sm:p-6 border border-border/50">
      <div className="flex flex-col sm:flex-row gap-4 sm:gap-6">
        <Skeleton className="h-32 sm:h-40 sm:w-56 rounded-lg shrink-0" />
        <div className="flex-1 space-y-4">
          <div className="space-y-2">
            <Skeleton className="h-6 w-3/4" />
            <Skeleton className="h-4 w-1/2" />
          </div>
          <div className="space-y-2">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-4 w-1/2" />
          </div>
          <div className="flex gap-2">
            <Skeleton className="h-6 w-20" />
            <Skeleton className="h-6 w-24" />
          </div>
        </div>
        <div className="flex flex-row sm:flex-col justify-between sm:justify-center items-end gap-4">
          <Skeleton className="h-8 w-28" />
          <Skeleton className="h-10 w-32" />
        </div>
      </div>
    </div>
  )
}

function getStatusConfig(status: string) {
  switch (status) {
    case "confirmed":
      return {
        label: "Confirmed",
        color: "bg-green-500/20 text-green-400 border-green-500/30",
        icon: CheckCircle2,
      }
    case "pending":
      return {
        label: "Pending Payment",
        color: "bg-amber-500/20 text-amber-400 border-amber-500/30",
        icon: Timer,
      }
    case "completed":
      return {
        label: "Completed",
        color: "bg-blue-500/20 text-blue-400 border-blue-500/30",
        icon: CheckCircle2,
      }
    case "cancelled":
      return {
        label: "Cancelled",
        color: "bg-red-500/20 text-red-400 border-red-500/30",
        icon: XCircle,
      }
    default:
      return {
        label: status,
        color: "bg-gray-500/20 text-gray-400 border-gray-500/30",
        icon: AlertCircle,
      }
  }
}

function BookingCard({ booking }: { booking: BookingWithEvent }) {
  const statusConfig = getStatusConfig(booking.status)
  const StatusIcon = statusConfig.icon

  return (
    <div className="group glass rounded-xl p-4 sm:p-6 border border-border/50 hover:border-primary/50 transition-all duration-300">
      <div className="flex flex-col sm:flex-row gap-4 sm:gap-6">
        {/* Event Image */}
        <div className="relative h-40 sm:h-40 sm:w-56 shrink-0 overflow-hidden rounded-lg">
          <img
            src={booking.event?.image || "/placeholder.svg"}
            alt={booking.event?.title || "Event"}
            className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
          />
          <div className="absolute top-2 right-2">
            <Badge className={`${statusConfig.color} border`}>
              <StatusIcon className="h-3 w-3 mr-1" />
              {statusConfig.label}
            </Badge>
          </div>
        </div>

        {/* Booking Info */}
        <div className="flex-1 space-y-3">
          <div>
            <h3 className="text-xl font-bold text-foreground group-hover:text-primary transition-colors line-clamp-1">
              {booking.event?.title || "Unknown Event"}
            </h3>
            <div className="flex items-center gap-2 text-muted-foreground text-sm mt-1">
              <MapPin className="h-4 w-4" />
              <span className="line-clamp-1">{booking.event?.venue || "TBA"}</span>
            </div>
          </div>

          <div className="flex flex-wrap gap-4 text-sm">
            <div className="flex items-center gap-2 text-muted-foreground">
              <Calendar className="h-4 w-4 text-primary" />
              <span>{booking.event?.date || "TBA"}</span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <Clock className="h-4 w-4 text-primary" />
              <span>{booking.event?.time || "TBA"}</span>
            </div>
          </div>

          <div className="flex flex-wrap items-center gap-3">
            <div className="flex items-center gap-2 text-sm">
              <Ticket className="h-4 w-4 text-primary" />
              <span className="text-foreground font-medium">{booking.tickets?.zone || "Standard"}</span>
              <span className="text-muted-foreground">× {booking.tickets?.quantity || 1}</span>
            </div>
          </div>

          <div className="text-xs text-muted-foreground">
            Booked on {new Date(booking.created_at).toLocaleDateString("en-US", { 
              year: "numeric", 
              month: "short", 
              day: "numeric" 
            })}
          </div>
        </div>

        {/* Price and Actions */}
        <div className="flex flex-row sm:flex-col justify-between sm:justify-between items-end sm:items-end gap-4 pt-4 sm:pt-0 border-t sm:border-t-0 border-border/50">
          <div className="text-right">
            <p className="text-xs text-muted-foreground">Total</p>
            <p className="text-2xl font-bold bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
              ฿{booking.total_amount.toLocaleString()}
            </p>
          </div>
          
          <div className="flex flex-col gap-2">
            {booking.status === "confirmed" && (
              <Button
                size="sm"
                className="bg-linear-to-r from-primary to-amber-400 hover:from-amber-400 hover:to-primary text-primary-foreground font-semibold"
              >
                <QrCode className="h-4 w-4 mr-2" />
                View Tickets
              </Button>
            )}
            {booking.status === "pending" && (
              <Button
                size="sm"
                className="bg-linear-to-r from-primary to-amber-400 hover:from-amber-400 hover:to-primary text-primary-foreground font-semibold"
              >
                Complete Payment
              </Button>
            )}
            {booking.status === "completed" && (
              <Button
                size="sm"
                variant="outline"
                className="border-primary/50 text-primary hover:bg-primary/10"
              >
                <Download className="h-4 w-4 mr-2" />
                Download
              </Button>
            )}
            <Link href={`/events/${booking.event?.id || ""}`}>
              <Button
                size="sm"
                variant="ghost"
                className="text-muted-foreground hover:text-primary w-full"
              >
                View Event
                <ChevronRight className="h-4 w-4 ml-1" />
              </Button>
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}

type StatusFilter = "all" | "confirmed" | "pending" | "completed" | "cancelled"

export default function MyBookingsPage() {
  const router = useRouter()
  const { isAuthenticated, isLoading: authLoading } = useAuth()
  const [bookings, setBookings] = useState<BookingWithEvent[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("all")
  const [searchQuery, setSearchQuery] = useState("")

  useEffect(() => {
    if (!authLoading && !isAuthenticated) {
      router.push("/login?redirect=/my-bookings")
      return
    }

    async function fetchBookings() {
      setIsLoading(true)
      try {
        // Try to fetch from API first
        const apiBookings = await bookingApi.listUserBookings()
        // If API returns empty, use mock data for demo
        if (apiBookings && apiBookings.length > 0) {
          setBookings(apiBookings as BookingWithEvent[])
        } else {
          setBookings(MOCK_BOOKINGS)
        }
      } catch (error) {
        console.warn("Failed to fetch bookings from API, using mock data:", error)
        setBookings(MOCK_BOOKINGS)
      } finally {
        setIsLoading(false)
      }
    }

    if (isAuthenticated) {
      fetchBookings()
    }
  }, [isAuthenticated, authLoading, router])

  // Filter bookings
  const filteredBookings = bookings.filter((booking) => {
    const matchesStatus = statusFilter === "all" || booking.status === statusFilter
    const matchesSearch =
      searchQuery === "" ||
      booking.event?.title?.toLowerCase().includes(searchQuery.toLowerCase()) ||
      booking.event?.venue?.toLowerCase().includes(searchQuery.toLowerCase())
    return matchesStatus && matchesSearch
  })

  // Group bookings by status for summary
  const bookingSummary = {
    total: bookings.length,
    confirmed: bookings.filter((b) => b.status === "confirmed").length,
    pending: bookings.filter((b) => b.status === "pending").length,
    completed: bookings.filter((b) => b.status === "completed").length,
    cancelled: bookings.filter((b) => b.status === "cancelled").length,
  }

  if (authLoading) {
    return (
      <main className="min-h-screen bg-background">
        <Header />
        <div className="container mx-auto px-4 lg:px-8 pt-24 pb-16">
          <div className="flex items-center justify-center h-64">
            <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary" />
          </div>
        </div>
      </main>
    )
  }

  return (
    <main className="min-h-screen bg-background">
      <Header />

      {/* Hero Section */}
      <section className="relative pt-24 pb-12 lg:pt-32 lg:pb-16 overflow-hidden">
        {/* Background */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-primary/20 via-background to-background" />
        <div className="absolute inset-0 bg-[url('/images/grid-pattern.svg')] opacity-5" />

        <div className="container mx-auto px-4 lg:px-8 relative z-10">
          <div className="max-w-3xl mx-auto text-center space-y-6">
            <div className="inline-block glass px-4 py-2 rounded-full">
              <span className="text-primary text-sm font-medium flex items-center gap-2">
                <Ticket className="h-4 w-4" />
                My Bookings
              </span>
            </div>
            <h1 className="text-4xl lg:text-5xl font-bold text-balance">
              Your{" "}
              <span className="bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                Tickets & Bookings
              </span>
            </h1>
            <p className="text-lg text-muted-foreground max-w-xl mx-auto text-pretty">
              Manage all your event bookings, view tickets, and track your upcoming experiences.
            </p>
          </div>
        </div>
      </section>

      {/* Stats Summary */}
      <section className="container mx-auto px-4 lg:px-8 -mt-4 mb-8">
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <div className="glass rounded-xl p-4 border border-border/50 text-center">
            <p className="text-3xl font-bold bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
              {bookingSummary.total}
            </p>
            <p className="text-sm text-muted-foreground">Total Bookings</p>
          </div>
          <div className="glass rounded-xl p-4 border border-green-500/30 text-center">
            <p className="text-3xl font-bold text-green-400">{bookingSummary.confirmed}</p>
            <p className="text-sm text-muted-foreground">Confirmed</p>
          </div>
          <div className="glass rounded-xl p-4 border border-amber-500/30 text-center">
            <p className="text-3xl font-bold text-amber-400">{bookingSummary.pending}</p>
            <p className="text-sm text-muted-foreground">Pending</p>
          </div>
          <div className="glass rounded-xl p-4 border border-blue-500/30 text-center">
            <p className="text-3xl font-bold text-blue-400">{bookingSummary.completed}</p>
            <p className="text-sm text-muted-foreground">Completed</p>
          </div>
        </div>
      </section>

      {/* Bookings Section */}
      <section className="container mx-auto px-4 lg:px-8 pb-16 lg:pb-24">
        {/* Filters */}
        <div className="flex flex-col sm:flex-row gap-4 mb-8">
          {/* Search */}
          <div className="relative flex-1">
            <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
            <Input
              type="text"
              placeholder="Search bookings..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-12 glass border-primary/30 focus:border-primary"
            />
          </div>

          {/* Status Filter */}
          <Select value={statusFilter} onValueChange={(v) => setStatusFilter(v as StatusFilter)}>
            <SelectTrigger className="w-full sm:w-48 border-primary/30">
              <Filter className="h-4 w-4 mr-2" />
              <SelectValue placeholder="Filter by status" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All Bookings</SelectItem>
              <SelectItem value="confirmed">Confirmed</SelectItem>
              <SelectItem value="pending">Pending</SelectItem>
              <SelectItem value="completed">Completed</SelectItem>
              <SelectItem value="cancelled">Cancelled</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {/* Bookings List */}
        {isLoading ? (
          <div className="space-y-4">
            <BookingCardSkeleton />
            <BookingCardSkeleton />
            <BookingCardSkeleton />
          </div>
        ) : filteredBookings.length > 0 ? (
          <div className="space-y-4">
            {filteredBookings.map((booking) => (
              <BookingCard key={booking.id} booking={booking} />
            ))}
          </div>
        ) : (
          <div className="text-center py-16 space-y-4">
            <div className="glass inline-block p-6 rounded-full">
              <Ticket className="h-12 w-12 text-muted-foreground" />
            </div>
            <h3 className="text-2xl font-semibold text-foreground">No bookings found</h3>
            <p className="text-muted-foreground max-w-md mx-auto">
              {statusFilter !== "all"
                ? `You don't have any ${statusFilter} bookings.`
                : "You haven't made any bookings yet. Start exploring events!"}
            </p>
            <Link href="/events">
              <Button className="mt-4 bg-linear-to-r from-primary to-amber-400 text-primary-foreground">
                Browse Events
              </Button>
            </Link>
          </div>
        )}
      </section>

      {/* Footer */}
      <footer className="glass border-t border-border/50">
        <div className="container mx-auto px-4 lg:px-8 py-12">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
            <div className="space-y-4">
              <div className="text-2xl font-bold bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                BookingRush
              </div>
              <p className="text-sm text-muted-foreground">
                Your premier destination for luxury event booking experiences.
              </p>
            </div>
            <div>
              <h3 className="font-semibold mb-4 text-foreground">Company</h3>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li>
                  <a href="#" className="hover:text-primary transition-colors">
                    About Us
                  </a>
                </li>
                <li>
                  <a href="#" className="hover:text-primary transition-colors">
                    Careers
                  </a>
                </li>
                <li>
                  <a href="#" className="hover:text-primary transition-colors">
                    Press
                  </a>
                </li>
              </ul>
            </div>
            <div>
              <h3 className="font-semibold mb-4 text-foreground">Support</h3>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li>
                  <a href="#" className="hover:text-primary transition-colors">
                    Help Center
                  </a>
                </li>
                <li>
                  <a href="#" className="hover:text-primary transition-colors">
                    Contact Us
                  </a>
                </li>
                <li>
                  <a href="#" className="hover:text-primary transition-colors">
                    FAQ
                  </a>
                </li>
              </ul>
            </div>
            <div>
              <h3 className="font-semibold mb-4 text-foreground">Legal</h3>
              <ul className="space-y-2 text-sm text-muted-foreground">
                <li>
                  <a href="#" className="hover:text-primary transition-colors">
                    Privacy Policy
                  </a>
                </li>
                <li>
                  <a href="#" className="hover:text-primary transition-colors">
                    Terms of Service
                  </a>
                </li>
                <li>
                  <a href="#" className="hover:text-primary transition-colors">
                    Cookie Policy
                  </a>
                </li>
              </ul>
            </div>
          </div>
          <div className="mt-12 pt-8 border-t border-border/50 text-center text-sm text-muted-foreground">
            <p>© 2025 BookingRush. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </main>
  )
}
