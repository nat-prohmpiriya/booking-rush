"use client"

import * as React from "react"
import { format } from "date-fns"
import { Calendar as CalendarIcon, Clock } from "lucide-react"

import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Calendar } from "@/components/ui/calendar"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"

interface DateTimePickerProps {
  value?: string // ISO string or datetime-local format (YYYY-MM-DDTHH:MM)
  onChange?: (value: string) => void
  placeholder?: string
  className?: string
  disabled?: boolean
}

// Parse date string to components (avoid timezone issues)
function parseDateTimeString(val: string | undefined): {
  year: number
  month: number
  day: number
  hour: string
  minute: string
} | null {
  if (!val) return null
  try {
    // Handle ISO string: 2024-12-28T10:00:00.000Z or datetime-local: 2024-12-28T10:00
    const cleanVal = val.replace("Z", "")
    const [datePart, timePart] = cleanVal.split("T")
    if (!datePart) return null

    const [year, month, day] = datePart.split("-").map(Number)
    if (!year || !month || !day) return null

    let hour = "00"
    let minute = "00"
    if (timePart) {
      const [h, m] = timePart.split(":")
      hour = h?.padStart(2, "0") || "00"
      minute = m?.padStart(2, "0") || "00"
    }

    return { year, month, day, hour, minute }
  } catch {
    return null
  }
}

// Create Date object from components (local time)
function createLocalDate(year: number, month: number, day: number): Date {
  return new Date(year, month - 1, day)
}

// Format to datetime-local string: YYYY-MM-DDTHH:MM
function formatDateTime(year: number, month: number, day: number, hour: string, minute: string): string {
  const y = year.toString()
  const m = String(month).padStart(2, "0")
  const d = String(day).padStart(2, "0")
  return `${y}-${m}-${d}T${hour}:${minute}`
}

export function DateTimePicker({
  value,
  onChange,
  placeholder = "Pick date and time",
  className,
  disabled = false,
}: DateTimePickerProps) {
  const [open, setOpen] = React.useState(false)

  const parsed = parseDateTimeString(value)

  const selectedDate = parsed ? createLocalDate(parsed.year, parsed.month, parsed.day) : undefined
  const selectedHour = parsed?.hour || "00"
  const selectedMinute = parsed?.minute || "00"

  // Generate hours (00-23)
  const hours = Array.from({ length: 24 }, (_, i) => i.toString().padStart(2, "0"))
  // Generate minutes (00, 15, 30, 45)
  const minutes = ["00", "15", "30", "45"]

  const handleDateSelect = (date: Date | undefined) => {
    if (!date) return
    const year = date.getFullYear()
    const month = date.getMonth() + 1
    const day = date.getDate()
    onChange?.(formatDateTime(year, month, day, selectedHour, selectedMinute))
  }

  const handleHourSelect = (hour: string) => {
    if (!parsed) {
      // If no date selected yet, use today
      const today = new Date()
      onChange?.(formatDateTime(today.getFullYear(), today.getMonth() + 1, today.getDate(), hour, selectedMinute))
    } else {
      onChange?.(formatDateTime(parsed.year, parsed.month, parsed.day, hour, selectedMinute))
    }
  }

  const handleMinuteSelect = (minute: string) => {
    if (!parsed) {
      // If no date selected yet, use today
      const today = new Date()
      onChange?.(formatDateTime(today.getFullYear(), today.getMonth() + 1, today.getDate(), selectedHour, minute))
    } else {
      onChange?.(formatDateTime(parsed.year, parsed.month, parsed.day, selectedHour, minute))
    }
  }

  const formatDisplayValue = () => {
    if (!selectedDate) return null
    const h = parseInt(selectedHour, 10)
    const ampm = h >= 12 ? "PM" : "AM"
    const displayHour = h % 12 || 12
    return `${format(selectedDate, "PPP")} ${displayHour}:${selectedMinute} ${ampm}`
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          disabled={disabled}
          className={cn(
            "w-full justify-start text-left font-normal bg-input border-input",
            !value && "text-muted-foreground",
            className
          )}
        >
          <CalendarIcon className="mr-2 h-4 w-4" />
          {selectedDate ? formatDisplayValue() : <span>{placeholder}</span>}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <div className="flex">
          {/* Calendar */}
          <div className="border-r">
            <Calendar
              mode="single"
              selected={selectedDate}
              onSelect={handleDateSelect}
              initialFocus
            />
          </div>
          {/* Time Picker */}
          <div className="flex flex-col">
            <div className="flex items-center gap-1 px-3 py-2 border-b">
              <Clock className="h-4 w-4 text-muted-foreground" />
              <span className="text-sm font-medium">Time</span>
            </div>
            <div className="flex flex-1">
              {/* Hours */}
              <div className="border-r">
                <div className="px-2 py-1 text-xs text-muted-foreground text-center border-b">
                  Hr
                </div>
                <div className="h-[200px] overflow-y-auto">
                  <div className="flex flex-col p-1">
                    {hours.map((hour) => (
                      <Button
                        key={hour}
                        variant={selectedHour === hour ? "default" : "ghost"}
                        size="sm"
                        className="h-7 w-10 justify-center text-xs"
                        onClick={() => handleHourSelect(hour)}
                      >
                        {hour}
                      </Button>
                    ))}
                  </div>
                </div>
              </div>
              {/* Minutes */}
              <div>
                <div className="px-2 py-1 text-xs text-muted-foreground text-center border-b">
                  Min
                </div>
                <div className="h-[200px] overflow-y-auto">
                  <div className="flex flex-col p-1">
                    {minutes.map((minute) => (
                      <Button
                        key={minute}
                        variant={selectedMinute === minute ? "default" : "ghost"}
                        size="sm"
                        className="h-7 w-10 justify-center text-xs"
                        onClick={() => handleMinuteSelect(minute)}
                      >
                        {minute}
                      </Button>
                    ))}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        {/* Footer with Done button */}
        <div className="border-t p-2 flex justify-end">
          <Button size="sm" onClick={() => setOpen(false)}>
            Done
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  )
}
