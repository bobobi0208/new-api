import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  deleteReconciliationChannelConfig,
  listReconciliationChannelConfigs,
  upsertReconciliationChannelConfig,
} from './api'
import {
  RECONCILIATION_UPSTREAM_TYPE,
  type ApiResponse,
  type ReconciliationChannelConfig,
  type ReconciliationChannelConfigFormData,
} from './types'

const EMPTY_FORM: ReconciliationChannelConfigFormData = {
  channel_id: 0,
  upstream_type: RECONCILIATION_UPSTREAM_TYPE.NEWAPI,
  enabled: true,
  base_url: '',
  note: '',
}

type Props = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ChannelConfigDialog({ open, onOpenChange }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [form, setForm] = useState<ReconciliationChannelConfigFormData>(EMPTY_FORM)

  const { data, isLoading } = useQuery({
    queryKey: ['reconciliation', 'channels'],
    queryFn: () => listReconciliationChannelConfigs(),
    enabled: open,
  })
  const configs = data?.data ?? []

  const upsertMutation = useMutation({
    mutationFn: upsertReconciliationChannelConfig,
    onSuccess: (res: ApiResponse) => {
      if (!res.success) {
        toast.error(res.message ?? t('Failed to save'))
        return
      }
      toast.success(t('Saved'))
      setForm(EMPTY_FORM)
      queryClient.invalidateQueries({ queryKey: ['reconciliation', 'channels'] })
      queryClient.invalidateQueries({ queryKey: ['reconciliation', 'latest'] })
    },
    onError: () => toast.error(t('Failed to save')),
  })

  const deleteMutation = useMutation({
    mutationFn: deleteReconciliationChannelConfig,
    onSuccess: (res: ApiResponse) => {
      if (!res.success) {
        toast.error(res.message ?? t('Failed to delete'))
        return
      }
      toast.success(t('Deleted'))
      queryClient.invalidateQueries({ queryKey: ['reconciliation', 'channels'] })
      queryClient.invalidateQueries({ queryKey: ['reconciliation', 'latest'] })
    },
  })

  const handleSubmit = () => {
    if (!form.channel_id || form.channel_id <= 0) {
      toast.error(t('channel_id is required'))
      return
    }
    upsertMutation.mutate(form)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>{t('Channel Reconciliation Configs')}</DialogTitle>
          <DialogDescription>
            {t('Mark which channels point at a known upstream so the reconciliation tasks know how to pull usage.')}
          </DialogDescription>
        </DialogHeader>

        <div className="grid grid-cols-1 gap-3 md:grid-cols-6 md:items-end">
          <div className="md:col-span-1">
            <Label>{t('Channel ID')}</Label>
            <Input
              type="number"
              value={form.channel_id || ''}
              onChange={(e) =>
                setForm({ ...form, channel_id: Number(e.target.value) || 0 })
              }
            />
          </div>
          <div className="md:col-span-1">
            <Label>{t('Upstream Type')}</Label>
            <Select
              value={form.upstream_type}
              onValueChange={(v) => setForm({ ...form, upstream_type: v ?? '' })}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value={RECONCILIATION_UPSTREAM_TYPE.NEWAPI}>
                  new-api
                </SelectItem>
                <SelectItem value={RECONCILIATION_UPSTREAM_TYPE.SUB2API}>
                  sub2api
                </SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="md:col-span-2">
            <Label>{t('Base URL (optional)')}</Label>
            <Input
              placeholder={t('Leave empty to use channel.base_url') ?? ''}
              value={form.base_url}
              onChange={(e) => setForm({ ...form, base_url: e.target.value })}
            />
          </div>
          <div className="md:col-span-1 flex items-center gap-2 pb-2">
            <Switch
              checked={form.enabled}
              onCheckedChange={(v) => setForm({ ...form, enabled: v })}
            />
            <Label>{t('Enabled')}</Label>
          </div>
          <div className="md:col-span-1">
            <Button onClick={handleSubmit} className="w-full">
              <Plus className="h-4 w-4 mr-1" />
              {t('Save')}
            </Button>
          </div>
        </div>

        <div className="max-h-[420px] overflow-auto border rounded-md">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Channel ID')}</TableHead>
                <TableHead>{t('Upstream')}</TableHead>
                <TableHead>{t('Base URL')}</TableHead>
                <TableHead>{t('Enabled')}</TableHead>
                <TableHead className="w-[80px]"></TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {isLoading && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    {t('Loading...')}
                  </TableCell>
                </TableRow>
              )}
              {!isLoading && configs.length === 0 && (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground">
                    {t('No channel configured for reconciliation yet')}
                  </TableCell>
                </TableRow>
              )}
              {configs.map((cfg: ReconciliationChannelConfig) => (
                <TableRow key={cfg.id}>
                  <TableCell>{cfg.channel_id}</TableCell>
                  <TableCell>
                    <Badge variant="outline">{cfg.upstream_type || '-'}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs">
                    {cfg.base_url || <span className="text-muted-foreground">channel default</span>}
                  </TableCell>
                  <TableCell>
                    <Badge variant={cfg.enabled ? 'default' : 'secondary'}>
                      {cfg.enabled ? t('Enabled') : t('Disabled')}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={() => deleteMutation.mutate(cfg.channel_id)}
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
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
