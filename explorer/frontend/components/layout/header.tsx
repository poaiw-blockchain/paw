'use client'

import Link from 'next/link'
import { usePathname } from 'next/navigation'
import { Moon, Sun, Menu, X, Globe, ExternalLink, Trophy, FileText } from 'lucide-react'
import { useTheme } from 'next-themes'
import { Button } from '@/components/ui/button'
import { SearchBar } from '@/components/search/search-bar'
import { cn } from '@/lib/utils'
import { useState } from 'react'

const navigation = [
  { name: 'Explorer', href: '/' },
  { name: 'Blocks', href: '/blocks' },
  { name: 'Transactions', href: '/transactions' },
  { name: 'Validators', href: '/validators' },
  { name: 'Governance', href: '/governance' },
  { name: 'Staking', href: '/staking' },
  { name: 'DEX', href: '/dex' },
  { name: 'Rich List', href: '/richlist' },
  { name: 'Oracle', href: '/oracle' },
  { name: 'Compute', href: '/compute' },
]

const API_DOCS_URL = process.env.NEXT_PUBLIC_FLASK_URL
  ? `${process.env.NEXT_PUBLIC_FLASK_URL}/api/docs`
  : 'http://localhost:5000/api/docs'

export function Header() {
  const pathname = usePathname()
  const { theme, setTheme } = useTheme()
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false)

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container mx-auto px-4">
        <div className="flex h-16 items-center justify-between">
          {/* Logo */}
          <Link href="/" className="flex items-center gap-2 font-bold text-xl">
            <Globe className="h-6 w-6 text-primary" />
            <span className="hidden sm:inline">PAW Chain Explorer</span>
            <span className="sm:hidden">PAW</span>
          </Link>

          {/* Desktop Navigation */}
          <nav className="hidden lg:flex items-center gap-1">
            {navigation.map((item) => (
              <Link key={item.name} href={item.href}>
                <Button
                  variant={pathname === item.href ? 'default' : 'ghost'}
                  size="sm"
                  className={cn('transition-colors')}
                >
                  {item.name}
                </Button>
              </Link>
            ))}
          </nav>

          {/* Search Bar (Desktop) */}
          <div className="hidden md:block flex-1 max-w-md mx-4">
            <SearchBar placeholder="Search..." />
          </div>

          {/* Theme Toggle, API Docs & Mobile Menu */}
          <div className="flex items-center gap-2">
            <a
              href={API_DOCS_URL}
              target="_blank"
              rel="noopener noreferrer"
              className="hidden sm:flex"
            >
              <Button variant="ghost" size="sm">
                <FileText className="h-4 w-4 mr-1" />
                API
                <ExternalLink className="h-3 w-3 ml-1" />
              </Button>
            </a>

            <Button
              variant="ghost"
              size="icon"
              onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}
            >
              <Sun className="h-5 w-5 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
              <Moon className="absolute h-5 w-5 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
              <span className="sr-only">Toggle theme</span>
            </Button>

            <Button
              variant="ghost"
              size="icon"
              className="lg:hidden"
              onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
            >
              {mobileMenuOpen ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
            </Button>
          </div>
        </div>

        {/* Mobile Search */}
        <div className="md:hidden pb-4">
          <SearchBar placeholder="Search..." />
        </div>

        {/* Mobile Navigation */}
        {mobileMenuOpen && (
          <div className="lg:hidden py-4 space-y-2 border-t">
            {navigation.map((item) => (
              <Link key={item.name} href={item.href} onClick={() => setMobileMenuOpen(false)}>
                <Button
                  variant={pathname === item.href ? 'default' : 'ghost'}
                  className="w-full justify-start"
                >
                  {item.name}
                </Button>
              </Link>
            ))}
            <a
              href={API_DOCS_URL}
              target="_blank"
              rel="noopener noreferrer"
              onClick={() => setMobileMenuOpen(false)}
            >
              <Button variant="ghost" className="w-full justify-start">
                <FileText className="h-4 w-4 mr-2" />
                API Documentation
                <ExternalLink className="h-3 w-3 ml-2" />
              </Button>
            </a>
          </div>
        )}
      </div>
    </header>
  )
}
