import { createFileRoute, redirect } from '@tanstack/react-router'
import { useAuthStore } from '@/stores/auth-store'
import { ROLE } from '@/lib/roles'
import { Sales } from '@/features/sales'

export const Route = createFileRoute('/_authenticated/sales/')({
  beforeLoad: () => {
    const { auth } = useAuthStore.getState()
    if (!auth.user || auth.user.role < ROLE.SALES) {
      throw redirect({ to: '/403' })
    }
  },
  component: Sales,
})
