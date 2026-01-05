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
    <header className="fixed top-0 w-full z-50 bg-gold-gradient shadow-md uppercase" data-testid="header">
      <nav className="container mx-auto px-4 lg:px-8" data-testid="header-nav">
        <div className="flex items-center justify-between h-14 lg:h-16">
          {/* Logo */}
          <Link href="/" className="flex items-center space-x-2" data-testid="header-logo">
            <span className="text-3xl">🎫</span>
            <div className="text-2xl font-bold text-black">
              Booking Rush
            </div>
          </Link>

          {/* Desktop Navigation */}
          <div className="hidden md:flex items-center space-x-8" data-testid="header-desktop-nav">
            <Link href="/events" className="text-black hover:text-gray-700 transition-colors font-bold text-lg" data-testid="header-events-link">
              Events
            </Link>
            <Link href="/about" className="text-black hover:text-gray-700 transition-colors font-bold text-lg" data-testid="header-about-link">
              About Us
            </Link>
            <Link href="/contact" className="text-black hover:text-gray-700 transition-colors font-bold text-lg" data-testid="header-contact-link">
              Contact
            </Link>
            {isAuthenticated ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    variant="outline"
                    className="border-black text-black hover:bg-black hover:text-primary bg-transparent"
                    data-testid="header-user-menu-button"
                  >
                    <User className="h-4 w-4 mr-2" />
                    {user?.name?.split(" ")[0] || "Account"}
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48 bg-black border-primary/30 uppercase" data-testid="header-user-menu">
                  <DropdownMenuItem asChild>
                    <Link href="/profile" className="flex items-center cursor-pointer text-primary font-bold hover:text-amber-300" data-testid="header-profile-link">
                      <User className="h-4 w-4 mr-2" />
                      Profile
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link href="/my-bookings" className="flex items-center cursor-pointer text-primary font-bold hover:text-amber-300" data-testid="header-my-bookings-link">
                      <Ticket className="h-4 w-4 mr-2" />
                      My Bookings
                    </Link>
                  </DropdownMenuItem>
                  {canAccessOrganizerDashboard && (
                    <>
                      <DropdownMenuSeparator className="bg-primary/30" />
                      <DropdownMenuItem asChild>
                        <Link href="/organizer" className="flex items-center cursor-pointer text-primary font-bold hover:text-amber-300" data-testid="header-organizer-link">
                          <Building2 className="h-4 w-4 mr-2" />
                          Organizer
                        </Link>
                      </DropdownMenuItem>
                    </>
                  )}
                  <DropdownMenuSeparator className="bg-primary/30" />
                  <DropdownMenuItem onClick={handleLogout} className="cursor-pointer text-red-500 font-bold hover:text-red-400" data-testid="header-logout-button">
                    <LogOut className="h-4 w-4 mr-2" />
                    Logout
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <Link href="/login" data-testid="header-login-link">
                <Button
                  variant="outline"
                  className="border-black text-black hover:bg-black hover:text-primary bg-transparent"
                  data-testid="header-login-button"
                >
                  Login
                </Button>
              </Link>
            )}
          </div>

          {/* Mobile Menu Button */}
          <Button
            variant="ghost"
            size="icon"
            className="md:hidden text-black"
            onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            data-testid="header-mobile-menu-button"
          >
            <Menu className="h-6 w-6" />
          </Button>
        </div>

        {/* Mobile Navigation */}
        {mobileMenuOpen && (
          <div className="md:hidden py-4 space-y-4 border-t border-black/20" data-testid="header-mobile-nav">
            <Link href="/events" className="block text-black hover:text-gray-700 transition-colors font-bold" data-testid="header-mobile-events-link">
              Events
            </Link>
            <Link href="/about" className="block text-black hover:text-gray-700 transition-colors font-bold" data-testid="header-mobile-about-link">
              About Us
            </Link>
            <Link href="/contact" className="block text-black hover:text-gray-700 transition-colors font-bold" data-testid="header-mobile-contact-link">
              Contact
            </Link>
            {isAuthenticated ? (
              <>
                <Link href="/profile" className="block text-black hover:text-gray-700 transition-colors font-medium" data-testid="header-mobile-profile-link">
                  Profile
                </Link>
                {canAccessOrganizerDashboard && (
                  <Link href="/organizer" className="block text-black hover:text-gray-700 transition-colors font-medium" data-testid="header-mobile-organizer-link">
                    Organizer
                  </Link>
                )}
                <Button
                  variant="outline"
                  onClick={handleLogout}
                  className="w-full border-red-800 text-red-800 hover:bg-red-800 hover:text-white bg-transparent"
                  data-testid="header-mobile-logout-button"
                >
                  Logout
                </Button>
              </>
            ) : (
              <Link href="/login" data-testid="header-mobile-login-link">
                <Button
                  variant="outline"
                  className="w-full border-black text-black hover:bg-black hover:text-primary bg-transparent"
                  data-testid="header-mobile-login-button"
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
