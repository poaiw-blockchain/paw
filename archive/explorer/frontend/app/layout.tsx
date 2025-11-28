import type { Metadata } from 'next'
import { Inter } from 'next/font/google'
import { Providers } from '@/components/providers'
import { Header } from '@/components/layout/header'
import { Footer } from '@/components/layout/footer'
import './globals.css'

const inter = Inter({ subsets: ['latin'] })

export const metadata: Metadata = {
  title: 'PAW Chain Explorer - Blockchain Explorer',
  description: 'Real-time blockchain data and analytics for the PAW Chain network',
  keywords: ['blockchain', 'explorer', 'paw chain', 'cryptocurrency', 'transactions', 'blocks'],
  authors: [{ name: 'PAW Chain Team' }],
  openGraph: {
    type: 'website',
    locale: 'en_US',
    url: 'https://explorer.pawchain.io',
    title: 'PAW Chain Explorer',
    description: 'Real-time blockchain data and analytics for the PAW Chain network',
    siteName: 'PAW Chain Explorer',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'PAW Chain Explorer',
    description: 'Real-time blockchain data and analytics for the PAW Chain network',
  },
  robots: {
    index: true,
    follow: true,
  },
}

export default function RootLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body className={inter.className}>
        <Providers>
          <div className="flex min-h-screen flex-col">
            <Header />
            <main className="flex-1">{children}</main>
            <Footer />
          </div>
        </Providers>
      </body>
    </html>
  )
}
