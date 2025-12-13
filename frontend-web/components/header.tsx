"use client"

import Link from "next/link"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Menu, User, LogOut, Ticket, Building2 } from "lucide-react"
import { useState } from "react"
import { useAuth } from "@/contexts/auth-context"

export function Header() {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)
  const { user, isAuthenticated, logout } = useAuth()
  const router = useRouter()

  const handleLogout = () => {
    logout()
    router.push("/")
  }

  // Check if user can access organizer dashboard
  const canAccessOrganizerDashboard = user?.role === "organizer" || user?.role === "admin" || user?.role === "super_admin"

  return (
    <header className="fixed top-0 w-full z-50 glass">
      <nav className="container mx-auto px-4 lg:px-8">
        <div className="flex items-center justify-between h-16 lg:h-20">
          {/* Logo */}
          <Link href="/" className="flex items-center space-x-2">
            <span className="text-3xl">🎫</span>
            <div className="text-2xl font-bold bg-linear-to-r from-primary to-amber-400 bg-clip-text text-transparent">
              BookingRush
            </div>
          </Link>

          {/* Desktop Navigation */}
          <div className="hidden md:flex items-center space-x-8">
            <Link href="/" className="text-foreground hover:text-primary transition-colors">
              Home
            </Link>
            <Link href="/events" className="text-foreground hover:text-primary transition-colors">
              Events
            </Link>
            {isAuthenticated && (
              <Link href="/my-bookings" className="text-foreground hover:text-primary transition-colors">
                My Bookings
              </Link>
            )}
            {isAuthenticated ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="outline"
                    className="border-primary text-primary hover:bg-primary hover:text-primary-foreground bg-transparent"
                  >
                    <User className="h-4 w-4 mr-2" />
                    {user?.name?.split(" ")[0] || "Account"}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48">
                  <DropdownMenuItem asChild>
                    <Link href="/profile" className="flex items-center cursor-pointer">
                      <User className="h-4 w-4 mr-2" />
                      Profile
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href="/my-bookings" className="flex items-center cursor-pointer">
                      <Ticket className="h-4 w-4 mr-2" />
                      My Bookings
                    </Link>
                  </DropdownMenuItem>
                  {canAccessOrganizerDashboard && (
                    <>
                      <DropdownMenuSeparator />
                      <DropdownMenuItem asChild>
                        <Link href="/organizer" className="flex items-center cursor-pointer">
                          <Building2 className="h-4 w-4 mr-2" />
                          Organizer
                        </Link>
                      </DropdownMenuItem>
                    </>
                  )}
                  <DropdownMenuSeparator />
                  <DropdownMenuItem onClick={handleLogout} className="cursor-pointer text-destructive">
                    <LogOut className="h-4 w-4 mr-2" />
                    Logout
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <Link href="/login">
                <Button
                  variant="outline"
                  className="border-primary text-primary hover:bg-primary hover:text-primary-foreground bg-transparent"
                >
                  Login
                </Button>
              </Link>
            )}
          </div>

          {/* Mobile Menu Button */}
          <button className="md:hidden text-foreground" onClick={() => setMobileMenuOpen(!mobileMenuOpen)}>
            <Menu className="h-6 w-6" />
          </button>
        </div>

        {/* Mobile Navigation */}
        {mobileMenuOpen && (
          <div className="md:hidden py-4 space-y-4">
            <Link href="/" className="block text-foreground hover:text-primary transition-colors">
              Home
            </Link>
            <Link href="/events" className="block text-foreground hover:text-primary transition-colors">
              Events
            </Link>
            {isAuthenticated && (
              <Link href="/my-bookings" className="block text-foreground hover:text-primary transition-colors">
                My Bookings
              </Link>
            )}
            {isAuthenticated ? (
              <>
                <Link href="/profile" className="block text-foreground hover:text-primary transition-colors">
                  Profile
                </Link>
                {canAccessOrganizerDashboard && (
                  <Link href="/organizer" className="block text-foreground hover:text-primary transition-colors">
                    Organizer
                  </Link>
                )}
                <Button
                  variant="outline"
                  onClick={handleLogout}
                  className="w-full border-destructive text-destructive hover:bg-destructive hover:text-destructive-foreground bg-transparent"
                >
                  Logout
                </Button>
              </>
            ) : (
              <Link href="/login">
                <Button
                  variant="outline"
                  className="w-full border-primary text-primary hover:bg-primary hover:text-primary-foreground bg-transparent"
                >
                  Login
                </Button>
              </Link>
            )}
          </div>
        )}
      </nav>
    </header>
  )
}
