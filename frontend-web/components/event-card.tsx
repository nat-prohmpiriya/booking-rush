import { Card, CardContent } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Calendar, MapPin } from "lucide-react"
import Link from "next/link"

interface EventCardProps {
  id: string | number
  title: string
  venue: string
  date: string
  price: number
  image: string
}

export function EventCard({ id, title, venue, date, price, image }: EventCardProps) {
  return (
    <Link href={`/events/${id}`} className="block">
      <Card className="group overflow-hidden glass border-border/50 hover:border-primary/50 transition-all duration-300 cursor-pointer">
        <div className="relative h-48 lg:h-56 overflow-hidden">
          <img
            src={image || "/placeholder.svg"}
            alt={title}
            className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
          />
          <div className="absolute top-3 right-3 glass px-3 py-1 rounded-full">
            <div className="flex items-center gap-1 text-primary text-sm font-semibold">
              <Calendar className="h-3 w-3" />
              <span>{date}</span>
            </div>
          </div>
        </div>
        <CardContent className="p-5 space-y-4">
          <div className="space-y-2">
            <h3 className="text-xl font-bold text-foreground text-balance line-clamp-2 group-hover:text-primary transition-colors">
              {title}
            </h3>
            <div className="flex items-center gap-2 text-muted-foreground text-sm">
              <MapPin className="h-4 w-4" />
              <span className="line-clamp-1">{venue}</span>
            </div>
          </div>
          <div className="flex items-center justify-between pt-2 border-t border-border/50">
            <div>
              <p className="text-xs text-muted-foreground">From</p>
              <p className="text-2xl font-bold bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
                ฿{price.toLocaleString()}
              </p>
            </div>
            <Button className="bg-linear-to-r from-primary to-amber-400 hover:from-amber-400 hover:to-primary text-primary-foreground font-semibold">
              Book Now
            </Button>
          </div>
        </CardContent>
      </Card>
    </Link>
  )
}
