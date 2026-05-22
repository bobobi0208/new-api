import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { ArrowDown, ArrowUp } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { demoteSalesToUser, promoteUserToSales } from '@/features/commission/api'
import { USER_ROLE } from '../../constants'
import type { User } from '../../types'

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: User
  onSuccess?: () => void
}

export function UserSalesDialog({ open, onOpenChange, user, onSuccess }: Props) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const isSales = user.role === USER_ROLE.SALES

  const close = () => {
    onSuccess?.()
    qc.invalidateQueries({ queryKey: ['users'] })
    onOpenChange(false)
  }

  const promoteMut = useMutation({
    mutationFn: () => promoteUserToSales(user.id),
    onSuccess: () => {
      toast.success(t('Promoted to sales'))
      close()
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : t('Promote failed')),
  })
  const demoteMut = useMutation({
    mutationFn: () => demoteSalesToUser(user.id),
    onSuccess: () => {
      toast.success(t('Demoted to regular user'))
      close()
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : t('Demote failed')),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>
            {t('Sales Role')} — {user.username}
          </DialogTitle>
          <DialogDescription>
            {isSales
              ? t('This user is a sales (role=5). Demoting will stop new commission accrual; bound customers stay bound until you unbind them via SQL or future tooling.')
              : t('Promote a regular user to sales. After promotion they get a /sales dashboard with their own invite link; customers signing up via that link are auto-bound to them.')}
          </DialogDescription>
        </DialogHeader>

        <div className="rounded-md border p-3">
          {isSales ? (
            <>
              <div className="text-sm font-medium mb-2">{t('Demote to regular user')}</div>
              <p className="text-xs text-muted-foreground mb-3">
                {t('Existing bindings via inviter_id stay intact, but new consume will not accrue commission.')}
              </p>
              <Button
                variant="destructive"
                onClick={() => demoteMut.mutate()}
                disabled={demoteMut.isPending}
              >
                <ArrowDown className="size-4" /> {t('Demote')}
              </Button>
            </>
          ) : (
            <>
              <div className="text-sm font-medium mb-2">{t('Promote to sales')}</div>
              <p className="text-xs text-muted-foreground mb-3">
                {t('User will gain access to /sales dashboard and start accruing commission from customers who sign up via their invite link.')}
              </p>
              <Button onClick={() => promoteMut.mutate()} disabled={promoteMut.isPending}>
                <ArrowUp className="size-4" /> {t('Promote')}
              </Button>
            </>
          )}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('Close')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
