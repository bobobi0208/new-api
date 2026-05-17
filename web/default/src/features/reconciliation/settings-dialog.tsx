import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import {
  getReconciliationSettings,
  updateReconciliationSettings,
} from './api'
import type { ApiResponse, ReconciliationSetting } from './types'

const EMPTY_SETTING: ReconciliationSetting = {
  enabled: true,
  balance_poll_interval_sec: 600,
  daily_job_hour: 2,
  abs_threshold_usd: 0.01,
  rel_threshold: 0.05,
  http_timeout_sec: 15,
  retain_days: 90,
}

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function SettingsDialog({ open, onOpenChange }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [draft, setDraft] = useState<ReconciliationSetting>(EMPTY_SETTING)

  const { data } = useQuery({
    queryKey: ['reconciliation', 'settings'],
    queryFn: () => getReconciliationSettings(),
    enabled: open,
  })

  useEffect(() => {
    if (data?.data) setDraft(data.data)
  }, [data])

  const updateMutation = useMutation({
    mutationFn: updateReconciliationSettings,
    onSuccess: (res: ApiResponse) => {
      if (!res.success) {
        toast.error(res.message ?? t('Failed to save'))
        return
      }
      toast.success(t('Saved'))
      queryClient.invalidateQueries({ queryKey: ['reconciliation', 'settings'] })
      onOpenChange(false)
    },
    onError: () => toast.error(t('Failed to save')),
  })

  const set = <K extends keyof ReconciliationSetting>(
    key: K,
    value: ReconciliationSetting[K]
  ) => setDraft((d) => ({ ...d, [key]: value }))

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('Reconciliation Settings')}</DialogTitle>
          <DialogDescription>
            {t('Global thresholds and schedule for upstream reconciliation tasks.')}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3">
          <div className="flex items-center justify-between">
            <Label>{t('Enable reconciliation tasks')}</Label>
            <Switch
              checked={draft.enabled}
              onCheckedChange={(v) => set('enabled', v)}
            />
          </div>
          <div className="grid gap-1">
            <Label>{t('Balance poll interval (seconds)')}</Label>
            <Input
              type="number"
              min={60}
              value={draft.balance_poll_interval_sec}
              onChange={(e) =>
                set('balance_poll_interval_sec', Number(e.target.value) || 60)
              }
            />
          </div>
          <div className="grid gap-1">
            <Label>{t('Daily job hour (0-23, local time)')}</Label>
            <Input
              type="number"
              min={0}
              max={23}
              value={draft.daily_job_hour}
              onChange={(e) => set('daily_job_hour', Number(e.target.value) || 0)}
            />
          </div>
          <div className="grid gap-1">
            <Label>{t('Absolute threshold (USD)')}</Label>
            <Input
              type="number"
              step="0.01"
              min={0}
              value={draft.abs_threshold_usd}
              onChange={(e) => set('abs_threshold_usd', Number(e.target.value) || 0)}
            />
          </div>
          <div className="grid gap-1">
            <Label>{t('Relative threshold (0-1)')}</Label>
            <Input
              type="number"
              step="0.01"
              min={0}
              max={1}
              value={draft.rel_threshold}
              onChange={(e) => set('rel_threshold', Number(e.target.value) || 0)}
            />
          </div>
          <div className="grid gap-1">
            <Label>{t('HTTP timeout (seconds)')}</Label>
            <Input
              type="number"
              min={3}
              max={120}
              value={draft.http_timeout_sec}
              onChange={(e) => set('http_timeout_sec', Number(e.target.value) || 15)}
            />
          </div>
          <div className="grid gap-1">
            <Label>{t('Retain days for old records')}</Label>
            <Input
              type="number"
              min={0}
              value={draft.retain_days}
              onChange={(e) => set('retain_days', Number(e.target.value) || 0)}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('Cancel')}
          </Button>
          <Button onClick={() => updateMutation.mutate(draft)} disabled={updateMutation.isPending}>
            {t('Save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
