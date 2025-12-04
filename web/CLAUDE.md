# Tikket Shop - Frontend Style Guide

## Design System

### Theme: Dark Premium + Urgency
สไตล์แอพจอง ticket/concert ที่ดูหรูหรา พร้อมสร้างความเร่งด่วนให้ user ตัดสินใจเร็ว

### Color Palette

| Color | CSS Variable | OKLCH Value | Usage |
|-------|--------------|-------------|-------|
| Background | `--background` | `oklch(0.13 0 0)` | พื้นหลังหลัก (เกือบดำ) |
| Foreground | `--foreground` | `oklch(0.98 0 0)` | ตัวอักษรหลัก (ขาว) |
| Card | `--card` | `oklch(0.18 0 0)` | Card background |
| Muted | `--muted` | `oklch(0.22 0 0)` | พื้นหลังรอง |
| Border | `--border` | `oklch(0.28 0 0)` | เส้นขอบ |
| Accent | `--accent` | `oklch(0.65 0.2 25)` | สีส้มแดง (urgency) |
| Success | `--success` | `oklch(0.65 0.18 145)` | สีเขียว |
| Warning | `--warning` | `oklch(0.75 0.18 85)` | สีเหลือง |
| Urgent | `--urgent` | `oklch(0.65 0.22 25)` | สีแดงส้ม (เร่งด่วน) |

### Zone Colors (Seat Pricing)

| Zone | Color Class | Price Tier |
|------|-------------|------------|
| VIP | `bg-amber-500` | สูงสุด |
| Premium | `bg-rose-500` | สูง |
| Standard | `bg-sky-500` | กลาง |
| Economy | `bg-emerald-500` | ประหยัด |

## Typography

### Font Family
- **Sans:** Geist (primary)
- **Mono:** Geist Mono (code, numbers)

### Font Sizes
```
text-xs    - 12px - labels, captions
text-sm    - 14px - secondary text
text-base  - 16px - body text
text-lg    - 18px - emphasized text
text-xl    - 20px - section headers
text-2xl   - 24px - page titles
text-3xl   - 30px - hero titles
```

### Font Weights
- `font-normal` - body text
- `font-medium` - labels, buttons
- `font-semibold` - headings
- `font-bold` - titles, emphasis

## Layout

### Container
```tsx
<div className="container mx-auto px-4">
```

### Responsive Breakpoints
- `sm:` - 640px
- `md:` - 768px
- `lg:` - 1024px
- `xl:` - 1280px

### Common Patterns
```tsx
// Two column with sidebar
<div className="grid gap-6 lg:grid-cols-[1fr_380px]">

// Sticky sidebar
<div className="lg:sticky lg:top-6 lg:self-start">

// Card grid
<div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
```

## Components

### Using shadcn/ui
ใช้ components จาก `@/components/ui/` เป็นหลัก

```tsx
import { Button } from "@/components/ui/button"
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
```

### Button Variants
```tsx
<Button>Primary</Button>
<Button variant="secondary">Secondary</Button>
<Button variant="outline">Outline</Button>
<Button variant="ghost">Ghost</Button>
<Button variant="destructive">Destructive</Button>
```

### Card Pattern
```tsx
<Card className="bg-card border-border">
  <CardHeader>
    <CardTitle>Title</CardTitle>
  </CardHeader>
  <CardContent>
    Content
  </CardContent>
</Card>
```

## Animations

### Urgency Pulse
ใช้เมื่อต้องการสร้างความเร่งด่วน (เช่น ที่นั่งใกล้หมด, countdown)
```tsx
<div className="animate-pulse-urgent">
  Only 5 seats left!
</div>
```

### Seat Hover
```tsx
<button className="seat-available">
  {/* Seat button */}
</button>
```

### Tailwind Animations
```tsx
// Fade in
<div className="animate-in fade-in">

// Slide in
<div className="animate-in slide-in-from-bottom">

// Pulse (built-in)
<div className="animate-pulse">
```

## Header Pattern

### Sticky Header with Blur
```tsx
<header className="sticky top-0 z-50 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
  <div className="container mx-auto flex h-14 items-center justify-between px-4">
    {/* Content */}
  </div>
</header>
```

### Step Indicator
```tsx
<div className="flex items-center gap-6 text-sm">
  <span className="text-muted-foreground">1. Event</span>
  <span className="text-muted-foreground">2. Tickets</span>
  <span className="font-medium text-foreground">3. Seats</span>
  <span className="text-muted-foreground">4. Payment</span>
</div>
```

## Page Structure

### Standard Page Layout
```tsx
export default function Page() {
  return (
    <div className="min-h-screen bg-background">
      <Header />

      <main className="container mx-auto px-4 py-6 lg:py-8">
        {/* Page title */}
        <div className="mb-6">
          <h1 className="text-2xl font-bold tracking-tight text-foreground lg:text-3xl">
            Page Title
          </h1>
          <p className="mt-1 text-muted-foreground">
            Description
          </p>
        </div>

        {/* Content */}
      </main>
    </div>
  )
}
```

## Naming Conventions

### Files
- Components: `kebab-case.tsx` (e.g., `event-card.tsx`)
- Pages: `page.tsx` (Next.js App Router)
- Hooks: `use-{name}.ts` (e.g., `use-booking.ts`)
- Utils: `{name}.ts` (e.g., `format-date.ts`)

### Components
- PascalCase: `EventCard`, `BookingHeader`
- Props interface: `{Component}Props`

```tsx
interface EventCardProps {
  event: Event
  onSelect?: (id: string) => void
}

export function EventCard({ event, onSelect }: EventCardProps) {
  // ...
}
```

## Best Practices

### Do's
- ใช้ CSS variables จาก theme (`text-foreground`, `bg-background`)
- ใช้ shadcn/ui components
- Responsive design (mobile-first)
- ใช้ `cn()` utility สำหรับ conditional classes
- ใช้ Lucide icons

### Don'ts
- ไม่ hardcode สี (ใช้ CSS variables)
- ไม่ใช้ inline styles
- ไม่สร้าง component ใหม่ถ้า shadcn/ui มีอยู่แล้ว
- ไม่ใช้ px units สำหรับ spacing (ใช้ Tailwind classes)

## Icon Usage

```tsx
import { ArrowLeft, Calendar, MapPin, Clock, Ticket } from "lucide-react"

<ArrowLeft className="h-5 w-5" />
<Calendar className="h-4 w-4 text-muted-foreground" />
```

## Mock Data Location

Mock data สำหรับ development อยู่ที่ `@/lib/mock/`
```
lib/
├── mock/
│   ├── events.ts
│   ├── shows.ts
│   └── bookings.ts
└── utils.ts
```
