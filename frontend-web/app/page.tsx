"use client"

import { useMemo } from "react"
import { Header } from "@/components/header"
import { Hero } from "@/components/hero"
import { EventSection } from "@/components/event-section"
import { useEvents, type EventDisplay } from "@/hooks/use-events"
import { Skeleton } from "@/components/ui/skeleton"

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

function LoadingSkeleton() {
  return (
    <div className="space-y-16">
      <section className="space-y-8">
        <div className="space-y-4">
          <Skeleton className="h-8 w-32 rounded-full" />
          <Skeleton className="h-10 w-64" />
          <Skeleton className="h-6 w-96" />
        </div>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 lg:gap-8">
          <EventCardSkeleton />
          <EventCardSkeleton />
          <EventCardSkeleton />
        </div>
      </section>
    </div>
  )
}

interface CategorizedEvents {
  onSale: EventDisplay[]
  comingSoon: EventDisplay[]
  pastEvents: EventDisplay[]
}

function categorizeEvents(events: EventDisplay[]): CategorizedEvents {
  const now = new Date()

  return events.reduce<CategorizedEvents>(
    (acc, event) => {
      const bookingStart = event.bookingStartAt ? new Date(event.bookingStartAt) : null
      const bookingEnd = event.bookingEndAt ? new Date(event.bookingEndAt) : null

      // Check if event is past (booking ended)
      if (bookingEnd && bookingEnd < now) {
        acc.pastEvents.push(event)
      }
      // Check if event is coming soon (booking not started yet)
      else if (bookingStart && bookingStart > now) {
        acc.comingSoon.push(event)
      }
      // Default: on sale (active or no specific booking window)
      else {
        acc.onSale.push(event)
      }

      return acc
    },
    { onSale: [], comingSoon: [], pastEvents: [] }
  )
}

export default function Home() {
  const { events, isLoading } = useEvents()

  const categorizedEvents = useMemo(() => categorizeEvents(events), [events])

  return (
    <main className="min-h-screen">
      <Header />
      <Hero />

      <div className="container mx-auto px-4 lg:px-8 py-16 lg:py-24 space-y-20">
        {isLoading ? (
          <LoadingSkeleton />
        ) : (
          <>
            {/* On Sale Events */}
            <EventSection
              badge="Hot"
              badgeVariant="primary"
              title="On Sale Now"
              subtitle="Get your tickets before they sell out!"
              events={categorizedEvents.onSale}
            />

            {/* Coming Soon Events */}
            <EventSection
              badge="Upcoming"
              badgeVariant="warning"
              title="Coming Soon"
              subtitle="Opening for booking soon. Stay tuned!"
              events={categorizedEvents.comingSoon}
            />

            {/* Past Events */}
            <EventSection
              badge="Archived"
              badgeVariant="muted"
              title="Past Events"
              subtitle="Events that have ended."
              events={categorizedEvents.pastEvents}
            />

            {/* Show message if no events at all */}
            {events.length === 0 && (
              <div className="text-center py-12">
                <p className="text-xl text-muted-foreground">
                  No events available at the moment.
                </p>
              </div>
            )}
          </>
        )}
      </div>
    </main>
  )
}
