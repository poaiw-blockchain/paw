import Link from 'next/link'
import { FileQuestion } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export default function NotFound() {
  return (
    <div className="container mx-auto py-16">
      <Card className="max-w-lg mx-auto">
        <CardHeader>
          <div className="flex items-center gap-3">
            <FileQuestion className="h-10 w-10 text-muted-foreground" />
            <div>
              <CardTitle className="text-2xl">404 - Page Not Found</CardTitle>
              <CardDescription>The page you are looking for does not exist</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <Link href="/">
            <Button className="w-full">Go Back Home</Button>
          </Link>
        </CardContent>
      </Card>
    </div>
  )
}
