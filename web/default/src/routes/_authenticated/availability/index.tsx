import { createFileRoute } from '@tanstack/react-router'
import { AvailabilityPage } from '@/features/availability'

export const Route = createFileRoute('/_authenticated/availability/')({
  component: AvailabilityPage,
})
