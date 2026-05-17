import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { PlayCircle, RefreshCcw, Settings, Workflow } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { SectionPageLayout } from '@/components/layout'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  getReconciliationRecord,
  listReconciliationAlerts,
  listReconciliationLatest,
  listReconciliationRecords,
  triggerReconciliation,
} from './api'
import { ChannelConfigDialog } from './channel-config-dialog'
import { SettingsDialog } from './settings-dialog'
import {
  RECONCILIATION_STATUS,
  type ApiResponse,
  type ReconciliationRecord,
} from './types'

const PAGE_SIZE = 50

function formatTs(ts?: number | null) {
  if (!ts || ts <= 0) return '-'
  return new Date(ts * 1000).toLocaleString()
}

function formatUSD(v?: number | null) {
  if (v === undefined || v === null || Number.isNaN(v)) return '-'
  return `$${v.toFixed(6)}`
}

function statusBadgeVariant(status: string) {
  switch (status) {
    case RECONCILIATION_STATUS.MATCH:
      return 'default' as const
    case RECONCILIATION_STATUS.MISMATCH:
      return 'destructive' as const
    case RECONCILIATION_STATUS.INCONCLUSIVE:
      return 'secondary' as const
    case RECONCILIATION_STATUS.ERROR:
      return 'destructive' as const
    default:
      return 'outline' as const
  }
}

export function Reconciliation() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const [tab, setTab] = useState<'latest' | 'records' | 'alerts'>('latest')
  const [channelConfigOpen, setChannelConfigOpen] = useState(false)
  const [settingsOpen, setSettingsOpen] = useState(false)
  const [detailRecordId, setDetailRecordId] = useState<number | null>(null)

  // Filters for Records tab
  const [recPage, setRecPage] = useState(1)
  const [recChannelId, setRecChannelId] = useState<string>('')
  const [recStatus, setRecStatus] = useState<string>('all')
  const [recRunType, setRecRunType] = useState<string>('all')

  const latestQuery = useQuery({
    queryKey: ['reconciliation', 'latest'],
    queryFn: () => listReconciliationLatest(),
    enabled: tab === 'latest',
  })
  const alertsQuery = useQuery({
    queryKey: ['reconciliation', 'alerts'],
    queryFn: () => listReconciliationAlerts(),
    enabled: tab === 'alerts',
  })
  const recordsQuery = useQuery({
    queryKey: ['reconciliation', 'records', recPage, recChannelId, recStatus, recRunType],
    queryFn: () =>
      listReconciliationRecords({
        p: recPage,
        page_size: PAGE_SIZE,
        channel_id: recChannelId ? Number(recChannelId) : undefined,
        status: recStatus !== 'all' ? recStatus : undefined,
        run_type: recRunType !== 'all' ? recRunType : undefined,
      }),
    enabled: tab === 'records',
  })

  const detailQuery = useQuery({
    queryKey: ['reconciliation', 'record', detailRecordId],
    queryFn: () =>
      detailRecordId ? getReconciliationRecord(detailRecordId) : Promise.resolve(null),
    enabled: !!detailRecordId,
  })

  const triggerMutation = useMutation({
    mutationFn: triggerReconciliation,
    onSuccess: (res: ApiResponse<ReconciliationRecord>) => {
      if (!res.success) {
        toast.error(res.message ?? t('Failed to trigger'))
        return
      }
      toast.success(t('Triggered'))
      queryClient.invalidateQueries({ queryKey: ['reconciliation'] })
    },
    onError: () => toast.error(t('Failed to trigger')),
  })

  const refreshAll = () => {
    queryClient.invalidateQueries({ queryKey: ['reconciliation'] })
  }

  const latest = latestQuery.data?.data ?? []
  const alerts = alertsQuery.data?.data ?? []
  const records = recordsQuery.data?.data?.items ?? []
  const recordsTotal = recordsQuery.data?.data?.total ?? 0
  const recordPageCount = useMemo(
    () => Math.max(1, Math.ceil(recordsTotal / PAGE_SIZE)),
    [recordsTotal]
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Upstream Reconciliation')}</SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button type="button" variant="outline" onClick={() => setChannelConfigOpen(true)}>
          <Workflow className="size-4" />
          {t('Channel Configs')}
        </Button>
        <Button type="button" variant="outline" onClick={() => setSettingsOpen(true)}>
          <Settings className="size-4" />
          {t('Settings')}
        </Button>
        <Button type="button" variant="outline" onClick={refreshAll}>
          <RefreshCcw className="size-4" />
          {t('Refresh')}
        </Button>
      </SectionPageLayout.Actions>

      <Tabs value={tab} onValueChange={(v) => setTab(v as typeof tab)} className="w-full">
        <TabsList>
          <TabsTrigger value="latest">{t('Latest per channel')}</TabsTrigger>
          <TabsTrigger value="alerts">
            {t('Alerts')}
            {alerts.length > 0 && (
              <Badge variant="destructive" className="ml-2">
                {alerts.length}
              </Badge>
            )}
          </TabsTrigger>
          <TabsTrigger value="records">{t('All records')}</TabsTrigger>
        </TabsList>

        <TabsContent value="latest">
          <RecordsTable
            records={latest}
            loading={latestQuery.isLoading}
            onTrigger={(channelId) =>
              triggerMutation.mutate({ channel_id: channelId })
            }
            onOpenDetail={(id) => setDetailRecordId(id)}
          />
        </TabsContent>

        <TabsContent value="alerts">
          <RecordsTable
            records={alerts}
            loading={alertsQuery.isLoading}
            onTrigger={(channelId) =>
              triggerMutation.mutate({ channel_id: channelId })
            }
            onOpenDetail={(id) => setDetailRecordId(id)}
            emptyHint={t('No mismatch alerts at the moment')}
          />
        </TabsContent>

        <TabsContent value="records">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-4 md:items-end mb-3">
            <div>
              <Label>{t('Channel ID')}</Label>
              <Input
                type="number"
                value={recChannelId}
                onChange={(e) => setRecChannelId(e.target.value)}
                placeholder={t('Any')}
              />
            </div>
            <div>
              <Label>{t('Status')}</Label>
              <Select value={recStatus} onValueChange={(v) => setRecStatus(v ?? 'all')}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{t('Any')}</SelectItem>
                  <SelectItem value={RECONCILIATION_STATUS.MATCH}>match</SelectItem>
                  <SelectItem value={RECONCILIATION_STATUS.MISMATCH}>mismatch</SelectItem>
                  <SelectItem value={RECONCILIATION_STATUS.INCONCLUSIVE}>
                    inconclusive
                  </SelectItem>
                  <SelectItem value={RECONCILIATION_STATUS.ERROR}>error</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div>
              <Label>{t('Run Type')}</Label>
              <Select value={recRunType} onValueChange={(v) => setRecRunType(v ?? 'all')}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">{t('Any')}</SelectItem>
                  <SelectItem value="balance_snapshot">balance_snapshot</SelectItem>
                  <SelectItem value="daily_diff">daily_diff</SelectItem>
                  <SelectItem value="manual">manual</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="flex gap-2">
              <Button onClick={() => setRecPage(1)} variant="outline">
                {t('Apply')}
              </Button>
              <Button
                onClick={() => {
                  setRecChannelId('')
                  setRecStatus('all')
                  setRecRunType('all')
                  setRecPage(1)
                }}
                variant="outline"
              >
                {t('Reset')}
              </Button>
            </div>
          </div>

          <RecordsTable
            records={records}
            loading={recordsQuery.isLoading}
            onTrigger={(channelId) =>
              triggerMutation.mutate({ channel_id: channelId })
            }
            onOpenDetail={(id) => setDetailRecordId(id)}
          />

          <div className="flex items-center justify-end gap-2 mt-3">
            <Button
              size="sm"
              variant="outline"
              disabled={recPage <= 1}
              onClick={() => setRecPage((p) => p - 1)}
            >
              {t('Previous')}
            </Button>
            <span className="text-sm text-muted-foreground">
              {recPage} / {recordPageCount}
            </span>
            <Button
              size="sm"
              variant="outline"
              disabled={recPage >= recordPageCount}
              onClick={() => setRecPage((p) => p + 1)}
            >
              {t('Next')}
            </Button>
          </div>
        </TabsContent>
      </Tabs>

      <ChannelConfigDialog
        open={channelConfigOpen}
        onOpenChange={setChannelConfigOpen}
      />
      <SettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />

      <Dialog
        open={!!detailRecordId}
        onOpenChange={(o) => !o && setDetailRecordId(null)}
      >
        <DialogContent className="max-w-2xl">
          <DialogHeader>
            <DialogTitle>{t('Record detail')}</DialogTitle>
            <DialogDescription>
              {t('Raw upstream response and parsed reconciliation result.')}
            </DialogDescription>
          </DialogHeader>
          {detailQuery.data?.data && (
            <div className="space-y-2 text-sm">
              <div>
                <strong>{t('Channel ID')}:</strong> {detailQuery.data.data.channel_id}
              </div>
              <div>
                <strong>{t('Upstream')}:</strong> {detailQuery.data.data.upstream_type}
              </div>
              <div>
                <strong>{t('Run Type')}:</strong> {detailQuery.data.data.run_type}
              </div>
              <div>
                <strong>{t('Status')}:</strong>{' '}
                <Badge variant={statusBadgeVariant(detailQuery.data.data.status)}>
                  {detailQuery.data.data.status}
                </Badge>
              </div>
              <div>
                <strong>{t('Upstream used USD')}:</strong>{' '}
                {formatUSD(detailQuery.data.data.upstream_used_usd)}
              </div>
              <div>
                <strong>{t('Local used USD')}:</strong>{' '}
                {formatUSD(detailQuery.data.data.local_used_usd)}
              </div>
              <div>
                <strong>{t('Delta USD')}:</strong>{' '}
                {formatUSD(detailQuery.data.data.delta_usd)}
              </div>
              <div>
                <strong>{t('Message')}:</strong>{' '}
                {detailQuery.data.data.message || '-'}
              </div>
              <div>
                <strong>{t('Raw response')}:</strong>
                <pre className="mt-1 max-h-[300px] overflow-auto bg-muted p-2 text-xs">
                  {detailQuery.data.data.upstream_raw_json || '-'}
                </pre>
              </div>
            </div>
          )}
          <DialogFooter>
            <Button variant="outline" onClick={() => setDetailRecordId(null)}>
              {t('Close')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </SectionPageLayout>
  )
}

type RecordsTableProps = {
  records: ReconciliationRecord[]
  loading: boolean
  onTrigger: (channelId: number) => void
  onOpenDetail: (id: number) => void
  emptyHint?: string
}

function RecordsTable({
  records,
  loading,
  onTrigger,
  onOpenDetail,
  emptyHint,
}: RecordsTableProps) {
  const { t } = useTranslation()
  return (
    <div className="border rounded-md">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead>{t('Channel')}</TableHead>
            <TableHead>{t('Upstream')}</TableHead>
            <TableHead>{t('Run Type')}</TableHead>
            <TableHead>{t('Status')}</TableHead>
            <TableHead>{t('Upstream USD')}</TableHead>
            <TableHead>{t('Local USD')}</TableHead>
            <TableHead>{t('Delta')}</TableHead>
            <TableHead>{t('Run At')}</TableHead>
            <TableHead className="w-[160px]"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {loading && (
            <TableRow>
              <TableCell colSpan={9} className="text-center text-muted-foreground">
                {t('Loading...')}
              </TableCell>
            </TableRow>
          )}
          {!loading && records.length === 0 && (
            <TableRow>
              <TableCell colSpan={9} className="text-center text-muted-foreground">
                {emptyHint ?? t('No records yet')}
              </TableCell>
            </TableRow>
          )}
          {records.map((r) => (
            <TableRow key={r.id}>
              <TableCell>{r.channel_id}</TableCell>
              <TableCell>
                <Badge variant="outline">{r.upstream_type}</Badge>
              </TableCell>
              <TableCell className="font-mono text-xs">{r.run_type}</TableCell>
              <TableCell>
                <Badge variant={statusBadgeVariant(r.status)}>{r.status}</Badge>
              </TableCell>
              <TableCell>{formatUSD(r.upstream_used_usd)}</TableCell>
              <TableCell>{formatUSD(r.local_used_usd)}</TableCell>
              <TableCell
                className={
                  Math.abs(r.delta_usd) > 0.0001
                    ? 'text-destructive'
                    : 'text-muted-foreground'
                }
              >
                {formatUSD(r.delta_usd)}
              </TableCell>
              <TableCell className="text-xs">{formatTs(r.run_at)}</TableCell>
              <TableCell>
                <div className="flex gap-1">
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => onOpenDetail(r.id)}
                  >
                    {t('Detail')}
                  </Button>
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={() => onTrigger(r.channel_id)}
                  >
                    <PlayCircle className="size-4" />
                  </Button>
                </div>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  )
}
