'use client'

import { BlockList } from '@/components/blocks/block-list'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'

export default function BlocksPage() {
  return (
    <div className="container mx-auto py-8">
      <Card className="mb-6">
        <CardHeader>
          <CardTitle className="text-3xl">All Blocks</CardTitle>
          <CardDescription>
            Browse all blocks on the PAW Chain network
          </CardDescription>
        </CardHeader>
      </Card>
      <BlockList
        variant="full"
        limit={20}
        showPagination={true}
        showSearch={true}
        showFilters={true}
        autoRefresh={true}
      />
    </div>
  )
}
