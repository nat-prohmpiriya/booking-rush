"use client"

import { useState } from "react"
import { Header } from "@/components/header"
import { EventCard } from "@/components/event-card"
import { useEvents } from "@/hooks/use-events"
import { Skeleton } from "@/components/ui/skeleton"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Search, Filter, Calendar, MapPin, Grid3X3, List, SlidersHorizontal } from "lucide-react"

function EventCardSkeleton() {
  return (
    <div className="rounded-lg border border-border/50 overflow-hidden">
      <Skeleton className="h-48 lg:h-56 w-full" />
      <div className="p-5 space-y-4">
        <div className="space-y-2">
          <Skeleton className="h-6 w-3/4" />
          <Skeleton className="h-4 w-1/2" />
        </div>
        <div className="flex items-center justify-between pt-2 border-t border-border/50">
          <div className="space-y-1">
            <Skeleton className="h-3 w-12" />
            <Skeleton className="h-8 w-20" />
          </div>
          <Skeleton className="h-10 w-24" />
        </div>
      </div>
    </div>
  )
}

function EventListSkeleton() {
  return (
    <div className="flex gap-6 p-4 rounded-lg border border-border/50">
      <Skeleton className="h-32 w-48 rounded-lg shrink-0" />
      <div className="flex-1 space-y-3">
        <Skeleton className="h-6 w-3/4" />
        <Skeleton className="h-4 w-1/2" />
        <Skeleton className="h-4 w-1/3" />
      </div>
      <div className="flex flex-col justify-between items-end">
        <Skeleton className="h-8 w-24" />
        <Skeleton className="h-10 w-28" />
      </div>
    </div>
  )
}

type ViewMode = "grid" | "list"

export default function EventsPage() {
  const { events, isLoading, total } = useEvents()
  const [searchQuery, setSearchQuery] = useState("")
  const [sortBy, setSortBy] = useState("date")
  const [viewMode, setViewMode] = useState<ViewMode>("grid")
  const [showFilters, setShowFilters] = useState(false)

  // Filter events based on search query
  const filteredEvents = events.filter((event) => {
    const matchesSearch =
      event.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      event.venue.toLowerCase().includes(searchQuery.toLowerCase())
    return matchesSearch
  })

  // Sort events
  const sortedEvents = [...filteredEvents].sort((a, b) => {
    switch (sortBy) {
      case "price-low":
        return a.price - b.price
      case "price-high":
        return b.price - a.price
      case "name":
        return a.title.localeCompare(b.title)
      default: // date
        return new Date(a.date).getTime() - new Date(b.date).getTime()
    }
  })

  return (
    <main className="min-h-screen bg-background">
      <Header />

      {/* Hero Section */}
      <section className="relative pt-24 pb-16 lg:pt-32 lg:pb-24 overflow-hidden">
        {/* Background */}
        <div className="absolute inset-0 bg-[radial-gradient(ellipse_at_top,var(--tw-gradient-stops))] from-primary/20 via-background to-background" />
        <div className="absolute inset-0 bg-[url('/images/grid-pattern.svg')] opacity-5" />

        <div className="container mx-auto px-4 lg:px-8 relative z-10">
          <div className="max-w-3xl mx-auto text-center space-y-6">
            <div className="inline-block glass px-4 py-2 rounded-full">
              <span className="text-primary text-sm font-medium flex items-center gap-2">
                <Calendar className="h-4 w-4" />
                Discover Events
              </span>
            </div>
            <h1 className="text-4xl lg:text-6xl font-bold text-balance">
              Find Your Next{" "}
              <span className="bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                Experience
              </span>
            </h1>
            <p className="text-lg text-muted-foreground max-w-xl mx-auto text-pretty">
              Browse through our curated selection of premium events. From concerts to exclusive experiences.
            </p>
          </div>

          {/* Search Bar */}
          <div className="max-w-2xl mx-auto mt-10">
            <div className="relative">
              <Search className="absolute left-4 top-1/2 -translate-y-1/2 h-5 w-5 text-muted-foreground" />
              <Input
                type="text"
                placeholder="Search events by name or venue..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-12 pr-4 h-14 text-lg glass border-primary/30 focus:border-primary placeholder:text-muted-foreground/60"
              />
            </div>
          </div>
        </div>
      </section>

      {/* Events Section */}
      <section className="container mx-auto px-4 lg:px-8 pb-16 lg:pb-24">
        {/* Filters and Controls */}
        <div className="flex flex-col lg:flex-row lg:items-center justify-between gap-4 mb-8">
          <div className="flex items-center gap-4">
            <p className="text-muted-foreground">
              <span className="text-foreground font-semibold">{sortedEvents.length}</span> events found
            </p>
          </div>

          <div className="flex items-center gap-3">
            {/* Filter Toggle Button */}
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowFilters(!showFilters)}
              className={`border-primary/50 ${showFilters ? "bg-primary/10 text-primary" : ""}`}
            >
              <SlidersHorizontal className="h-4 w-4 mr-2" />
              Filters
            </Button>

            {/* Sort Select */}
            <Select value={sortBy} onValueChange={setSortBy}>
              <SelectTrigger className="w-40 border-primary/30">
                <Filter className="h-4 w-4 mr-2" />
                <SelectValue placeholder="Sort by" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="date">Date</SelectItem>
                <SelectItem value="price-low">Price: Low to High</SelectItem>
                <SelectItem value="price-high">Price: High to Low</SelectItem>
                <SelectItem value="name">Name</SelectItem>
              </SelectContent>
            </Select>

            {/* View Mode Toggle */}
            <div className="hidden sm:flex items-center border border-border/50 rounded-lg p-1">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setViewMode("grid")}
                className={viewMode === "grid" ? "bg-primary/20 text-primary" : "text-muted-foreground"}
              >
                <Grid3X3 className="h-4 w-4" />
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setViewMode("list")}
                className={viewMode === "list" ? "bg-primary/20 text-primary" : "text-muted-foreground"}
              >
                <List className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>

        {/* Expandable Filters Panel */}
        {showFilters && (
          <div className="glass rounded-xl p-6 mb-8 space-y-4 animate-in fade-in slide-in-from-top-2 duration-200">
            <h3 className="font-semibold text-foreground">Filter by</h3>
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
              <div className="space-y-2">
                <label className="text-sm text-muted-foreground">Category</label>
                <Select>
                  <SelectTrigger className="border-primary/30">
                    <SelectValue placeholder="All Categories" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Categories</SelectItem>
                    <SelectItem value="concert">Concerts</SelectItem>
                    <SelectItem value="sports">Sports</SelectItem>
                    <SelectItem value="theater">Theater</SelectItem>
                    <SelectItem value="festival">Festivals</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <label className="text-sm text-muted-foreground">Date Range</label>
                <Select>
                  <SelectTrigger className="border-primary/30">
                    <SelectValue placeholder="Any Date" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Any Date</SelectItem>
                    <SelectItem value="today">Today</SelectItem>
                    <SelectItem value="week">This Week</SelectItem>
                    <SelectItem value="month">This Month</SelectItem>
                    <SelectItem value="year">This Year</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <label className="text-sm text-muted-foreground">Price Range</label>
                <Select>
                  <SelectTrigger className="border-primary/30">
                    <SelectValue placeholder="Any Price" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">Any Price</SelectItem>
                    <SelectItem value="free">Free</SelectItem>
                    <SelectItem value="0-500">฿0 - ฿500</SelectItem>
                    <SelectItem value="500-2000">฿500 - ฿2,000</SelectItem>
                    <SelectItem value="2000+">฿2,000+</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <label className="text-sm text-muted-foreground">Location</label>
                <Select>
                  <SelectTrigger className="border-primary/30">
                    <SelectValue placeholder="All Locations" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All Locations</SelectItem>
                    <SelectItem value="bangkok">Bangkok</SelectItem>
                    <SelectItem value="chiang-mai">Chiang Mai</SelectItem>
                    <SelectItem value="phuket">Phuket</SelectItem>
                    <SelectItem value="pattaya">Pattaya</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="flex justify-end gap-3 pt-2">
              <Button variant="outline" size="sm" className="border-primary/50">
                Clear All
              </Button>
              <Button size="sm" className="bg-linear-to-r from-primary to-amber-400 text-primary-foreground">
                Apply Filters
              </Button>
            </div>
          </div>
        )}

        {/* Events Display */}
        {isLoading ? (
          viewMode === "grid" ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 lg:gap-8">
              {Array.from({ length: 6 }).map((_, i) => (
                <EventCardSkeleton key={i} />
              ))}
            </div>
          ) : (
            <div className="space-y-4">
              {Array.from({ length: 4 }).map((_, i) => (
                <EventListSkeleton key={i} />
              ))}
            </div>
          )
        ) : sortedEvents.length > 0 ? (
          viewMode === "grid" ? (
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 lg:gap-8">
              {sortedEvents.map((event) => (
                <EventCard
                  key={event.id}
                  id={event.id}
                  title={event.title}
                  venue={event.venue}
                  date={event.date}
                  price={event.price}
                  image={event.image}
                />
              ))}
            </div>
          ) : (
            <div className="space-y-4">
              {sortedEvents.map((event) => (
                <EventListCard
                  key={event.id}
                  id={event.id}
                  title={event.title}
                  subtitle={event.subtitle}
                  venue={event.venue}
                  date={event.date}
                  price={event.price}
                  image={event.image}
                />
              ))}
            </div>
          )
        ) : (
          <div className="text-center py-16 space-y-4">
            <div className="glass inline-block p-6 rounded-full">
              <Search className="h-12 w-12 text-muted-foreground" />
            </div>
            <h3 className="text-2xl font-semibold text-foreground">No events found</h3>
            <p className="text-muted-foreground max-w-md mx-auto">
              We couldn&apos;t find any events matching your search. Try adjusting your filters or search terms.
            </p>
            <Button
              onClick={() => {
                setSearchQuery("")
                setSortBy("date")
              }}
              className="mt-4 bg-linear-to-r from-primary to-amber-400 text-primary-foreground"
            >
              Clear Search
            </Button>
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

// List View Card Component
interface EventListCardProps {
  id: string | number
  title: string
  subtitle?: string
  venue: string
  date: string
  price: number
  image: string
}

function EventListCard({ id, title, subtitle, venue, date, price, image }: EventListCardProps) {
  return (
    <div className="group flex flex-col sm:flex-row gap-4 sm:gap-6 p-4 glass rounded-xl border border-border/50 hover:border-primary/50 transition-all duration-300">
      {/* Image */}
      <div className="relative h-48 sm:h-32 sm:w-48 shrink-0 overflow-hidden rounded-lg">
        <img
          src={image || "/placeholder.svg"}
          alt={title}
          className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
        />
        <div className="absolute top-2 right-2 glass px-2 py-1 rounded-full">
          <div className="flex items-center gap-1 text-primary text-xs font-semibold">
            <Calendar className="h-3 w-3" />
            <span>{date}</span>
          </div>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 flex flex-col justify-between">
        <div className="space-y-2">
          <h3 className="text-xl font-bold text-foreground group-hover:text-primary transition-colors line-clamp-1">
            {title}
          </h3>
          {subtitle && <p className="text-sm text-muted-foreground line-clamp-1">{subtitle}</p>}
          <div className="flex items-center gap-2 text-muted-foreground text-sm">
            <MapPin className="h-4 w-4" />
            <span className="line-clamp-1">{venue}</span>
          </div>
        </div>
      </div>

      {/* Price and CTA */}
      <div className="flex sm:flex-col items-center sm:items-end justify-between sm:justify-center gap-4 pt-4 sm:pt-0 border-t sm:border-t-0 border-border/50">
        <div className="text-right">
          <p className="text-xs text-muted-foreground">From</p>
          <p className="text-2xl font-bold bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
            ฿{price.toLocaleString()}
          </p>
        </div>
        <a href={`/events/${id}`}>
          <Button className="bg-linear-to-r from-primary to-amber-400 hover:from-amber-400 hover:to-primary text-primary-foreground font-semibold">
            Book Now
          </Button>
        </a>
      </div>
    </div>
  )
}
