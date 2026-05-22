import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { CheckCircle2, PlayCircle, RefreshCcw, Settings2, X, Wallet } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
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
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { SectionPageLayout } from '@/components/layout'
import {
  approveWithdraw,
  confirmBill,
  generateMonthlyBills,
  getDefaultTiers,
  listAdminBills,
  listAdminWithdraws,
  listAllSales,
  markWithdrawPaid,
  rejectWithdraw,
  setDefaultTiers,
  updateSalesTiers,
} from './api'
import {
  BILL_STATUS,
  formatBP,
  formatTs,
  formatUSD,
  QUOTA_PER_USD,
  WITHDRAW_STATUS,
  type Tier,
} from '../sales/types'

const PAGE_SIZE = 20

function lastMonth() {
  const d = new Date()
  d.setDate(1)
  d.setMonth(d.getMonth() - 1)
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

function billStatusBadge(status: number) {
  return status === BILL_STATUS.CONFIRMED ? (
    <Badge>Confirmed</Badge>
  ) : (
    <Badge variant="secondary">Draft</Badge>
  )
}

function withdrawStatusBadge(status: number) {
  switch (status) {
    case WITHDRAW_STATUS.PAID:
      return <Badge>Paid</Badge>
    case WITHDRAW_STATUS.APPROVED:
      return <Badge variant="default">Approved</Badge>
    case WITHDRAW_STATUS.REJECTED:
      return <Badge variant="destructive">Rejected</Badge>
    default:
      return <Badge variant="secondary">Pending</Badge>
  }
}

export function Commission() {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const [tab, setTab] = useState<'bills' | 'withdraws' | 'sales-users' | 'default-tiers'>('bills')

  // Bills
  const [billMonth, setBillMonth] = useState<string>(lastMonth())
  const [billStatus, setBillStatus] = useState<number>(-1)
  const [billPage, setBillPage] = useState(1)
  const [generateMonth, setGenerateMonth] = useState<string>(lastMonth())
  const [generateOpen, setGenerateOpen] = useState(false)

  // Withdraws
  const [wfStatus, setWfStatus] = useState<number>(-1)
  const [wfPage, setWfPage] = useState(1)
  const [rejectFor, setRejectFor] = useState<number | null>(null)
  const [rejectReason, setRejectReason] = useState('')
  const [payFor, setPayFor] = useState<number | null>(null)
  const [payNote, setPayNote] = useState('')

  // Sales users
  const [salesPage, setSalesPage] = useState(1)
  const [editTierFor, setEditTierFor] = useState<number | null>(null)
  const [editTierRate, setEditTierRate] = useState<string>('0')
  const [editTierJSON, setEditTierJSON] = useState<string>('')

  // Default tiers
  const [defaultJSON, setDefaultJSON] = useState<string>('')

  const billsQuery = useQuery({
    queryKey: ['commission', 'bills', billMonth, billStatus, billPage],
    queryFn: () =>
      listAdminBills({
        month: billMonth || undefined,
        status: billStatus < 0 ? undefined : billStatus,
        page: billPage,
        page_size: PAGE_SIZE,
      }),
    enabled: tab === 'bills',
  })
  const withdrawsQuery = useQuery({
    queryKey: ['commission', 'withdraws', wfStatus, wfPage],
    queryFn: () =>
      listAdminWithdraws({
        status: wfStatus < 0 ? undefined : wfStatus,
        page: wfPage,
        page_size: PAGE_SIZE,
      }),
    enabled: tab === 'withdraws',
  })
  const salesQuery = useQuery({
    queryKey: ['commission', 'sales-users', salesPage],
    queryFn: () => listAllSales({ page: salesPage, page_size: PAGE_SIZE }),
    enabled: tab === 'sales-users',
  })
  const defaultsQuery = useQuery({
    queryKey: ['commission', 'default-tiers'],
    queryFn: getDefaultTiers,
    enabled: tab === 'default-tiers',
  })

  const generateMut = useMutation({
    mutationFn: generateMonthlyBills,
    onSuccess: (res) => {
      toast.success(
        t('Bills generated: created {{c}}, skipped {{s}}', {
          c: res.data?.created ?? 0,
          s: res.data?.skipped ?? 0,
        }),
      )
      setGenerateOpen(false)
      qc.invalidateQueries({ queryKey: ['commission', 'bills'] })
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : t('Generate failed')),
  })

  const confirmMut = useMutation({
    mutationFn: confirmBill,
    onSuccess: () => {
      toast.success(t('Bill confirmed'))
      qc.invalidateQueries({ queryKey: ['commission'] })
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : t('Confirm failed')),
  })

  const approveMut = useMutation({
    mutationFn: approveWithdraw,
    onSuccess: () => {
      toast.success(t('Approved'))
      qc.invalidateQueries({ queryKey: ['commission', 'withdraws'] })
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : t('Approve failed')),
  })

  const rejectMut = useMutation({
    mutationFn: (input: { id: number; reason: string }) =>
      rejectWithdraw(input.id, input.reason),
    onSuccess: () => {
      toast.success(t('Rejected'))
      setRejectFor(null)
      setRejectReason('')
      qc.invalidateQueries({ queryKey: ['commission', 'withdraws'] })
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : t('Reject failed')),
  })

  const paidMut = useMutation({
    mutationFn: (input: { id: number; note: string }) =>
      markWithdrawPaid(input.id, input.note),
    onSuccess: () => {
      toast.success(t('Marked as paid'))
      setPayFor(null)
      setPayNote('')
      qc.invalidateQueries({ queryKey: ['commission', 'withdraws'] })
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : t('Mark paid failed')),
  })

  const updateTierMut = useMutation({
    mutationFn: (input: { userId: number; rate: number; tiers: Tier[] }) =>
      updateSalesTiers(input.userId, {
        commission_rate: input.rate,
        commission_tier_config: input.tiers,
      }),
    onSuccess: () => {
      toast.success(t('Saved'))
      setEditTierFor(null)
      qc.invalidateQueries({ queryKey: ['commission', 'sales-users'] })
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : t('Save failed')),
  })

  const setDefaultsMut = useMutation({
    mutationFn: setDefaultTiers,
    onSuccess: () => {
      toast.success(t('Saved'))
      qc.invalidateQueries({ queryKey: ['commission', 'default-tiers'] })
    },
    onError: (e: unknown) =>
      toast.error(e instanceof Error ? e.message : t('Save failed')),
  })

  const bills = billsQuery.data?.data
  const withdraws = withdrawsQuery.data?.data
  const salesUsers = salesQuery.data?.data
  const defaultTiers = defaultsQuery.data?.data ?? []

  const billsPageCount = useMemo(
    () => Math.max(1, Math.ceil((bills?.total ?? 0) / PAGE_SIZE)),
    [bills?.total],
  )
  const wfPageCount = useMemo(
    () => Math.max(1, Math.ceil((withdraws?.total ?? 0) / PAGE_SIZE)),
    [withdraws?.total],
  )
  const salesPageCount = useMemo(
    () => Math.max(1, Math.ceil((salesUsers?.total ?? 0) / PAGE_SIZE)),
    [salesUsers?.total],
  )

  const openEditTier = (
    userId: number,
    rateBP: number,
    tierConfig: string | undefined,
  ) => {
    setEditTierFor(userId)
    setEditTierRate(String(rateBP))
    setEditTierJSON(tierConfig ?? '')
  }

  const submitEditTier = () => {
    let tiers: Tier[] = []
    const trimmed = editTierJSON.trim()
    if (trimmed) {
      try {
        const parsed = JSON.parse(trimmed)
        if (Array.isArray(parsed)) tiers = parsed
      } catch {
        toast.error(t('Invalid tier JSON'))
        return
      }
    }
    const rate = Number(editTierRate)
    if (!Number.isFinite(rate) || rate < 0 || rate > 2000) {
      toast.error(t('Rate must be 0-2000 bp (0-20%)'))
      return
    }
    updateTierMut.mutate({ userId: editTierFor!, rate, tiers })
  }

  const submitDefaults = () => {
    let tiers: Tier[] = []
    try {
      tiers = JSON.parse(defaultJSON)
      if (!Array.isArray(tiers)) throw new Error('not array')
    } catch {
      toast.error(t('Invalid JSON'))
      return
    }
    setDefaultsMut.mutate(tiers)
  }

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Commission Management')}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Review monthly settlement bills and withdraw requests, manage sales tiers.')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Actions>
        <Button variant="outline" onClick={() => qc.invalidateQueries({ queryKey: ['commission'] })}>
          <RefreshCcw className="size-4" /> {t('Refresh')}
        </Button>
        <Button onClick={() => setGenerateOpen(true)}>
          <PlayCircle className="size-4" />
          {t('Generate Monthly Bills')}
        </Button>
      </SectionPageLayout.Actions>

      <SectionPageLayout.Content>
        <Tabs value={tab} onValueChange={(v) => setTab(v as typeof tab)}>
          <TabsList>
            <TabsTrigger value="bills">{t('Bills')}</TabsTrigger>
            <TabsTrigger value="withdraws">
              {t('Withdraws')}
              {(withdraws?.items?.filter((w) => w.status === WITHDRAW_STATUS.PENDING).length ?? 0) >
                0 && (
                <Badge variant="destructive" className="ml-2">
                  {withdraws?.items?.filter((w) => w.status === WITHDRAW_STATUS.PENDING).length}
                </Badge>
              )}
            </TabsTrigger>
            <TabsTrigger value="sales-users">{t('Sales Users')}</TabsTrigger>
            <TabsTrigger value="default-tiers">{t('Default Tiers')}</TabsTrigger>
          </TabsList>

          <TabsContent value="bills" className="mt-4">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-3 mb-3">
              <div>
                <Label>{t('Month (YYYY-MM)')}</Label>
                <Input
                  type="month"
                  value={billMonth}
                  onChange={(e) => {
                    setBillMonth(e.target.value)
                    setBillPage(1)
                  }}
                />
              </div>
              <div>
                <Label>{t('Status')}</Label>
                <select
                  className="block w-full h-9 rounded-md border border-input bg-background px-3 text-sm"
                  value={billStatus}
                  onChange={(e) => {
                    setBillStatus(Number(e.target.value))
                    setBillPage(1)
                  }}
                >
                  <option value={-1}>{t('Any')}</option>
                  <option value={BILL_STATUS.DRAFT}>Draft</option>
                  <option value={BILL_STATUS.CONFIRMED}>Confirmed</option>
                </select>
              </div>
            </div>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>{t('Sales User')}</TableHead>
                  <TableHead>{t('Month')}</TableHead>
                  <TableHead className="text-right">{t('Total Consume')}</TableHead>
                  <TableHead className="text-right">{t('Commission')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Created')}</TableHead>
                  <TableHead>{t('Confirmed')}</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {bills?.items?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={9} className="text-center text-muted-foreground">
                      {t('No bills')}
                    </TableCell>
                  </TableRow>
                )}
                {bills?.items?.map((b) => (
                  <TableRow key={b.id}>
                    <TableCell>{b.id}</TableCell>
                    <TableCell>#{b.sales_user_id}</TableCell>
                    <TableCell>{b.year_month}</TableCell>
                    <TableCell className="text-right">{formatUSD(b.total_consume_quota)}</TableCell>
                    <TableCell className="text-right">{formatUSD(b.commission_quota)}</TableCell>
                    <TableCell>{billStatusBadge(b.status)}</TableCell>
                    <TableCell>{formatTs(b.created_at)}</TableCell>
                    <TableCell>{formatTs(b.confirmed_at)}</TableCell>
                    <TableCell>
                      {b.status === BILL_STATUS.DRAFT && (
                        <Button
                          size="sm"
                          onClick={() => confirmMut.mutate(b.id)}
                          disabled={confirmMut.isPending}
                        >
                          <CheckCircle2 className="size-4" /> {t('Confirm')}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={billPage}
              pageCount={billsPageCount}
              onChange={setBillPage}
              total={bills?.total ?? 0}
            />
          </TabsContent>

          <TabsContent value="withdraws" className="mt-4">
            <div className="mb-3">
              <Label>{t('Status')}</Label>
              <select
                className="block w-full h-9 rounded-md border border-input bg-background px-3 text-sm md:w-48"
                value={wfStatus}
                onChange={(e) => {
                  setWfStatus(Number(e.target.value))
                  setWfPage(1)
                }}
              >
                <option value={-1}>{t('Any')}</option>
                <option value={WITHDRAW_STATUS.PENDING}>Pending</option>
                <option value={WITHDRAW_STATUS.APPROVED}>Approved</option>
                <option value={WITHDRAW_STATUS.REJECTED}>Rejected</option>
                <option value={WITHDRAW_STATUS.PAID}>Paid</option>
              </select>
            </div>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>{t('Sales')}</TableHead>
                  <TableHead className="text-right">{t('Amount')}</TableHead>
                  <TableHead>{t('Status')}</TableHead>
                  <TableHead>{t('Applied')}</TableHead>
                  <TableHead>{t('Note')}</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {withdraws?.items?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground">
                      {t('No requests')}
                    </TableCell>
                  </TableRow>
                )}
                {withdraws?.items?.map((w) => (
                  <TableRow key={w.id}>
                    <TableCell>{w.id}</TableCell>
                    <TableCell>#{w.sales_user_id}</TableCell>
                    <TableCell className="text-right">{formatUSD(w.amount_quota)}</TableCell>
                    <TableCell>{withdrawStatusBadge(w.status)}</TableCell>
                    <TableCell>{formatTs(w.applied_at)}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {w.applicant_note ?? '-'}
                    </TableCell>
                    <TableCell className="space-x-1">
                      {w.status === WITHDRAW_STATUS.PENDING && (
                        <>
                          <Button size="sm" onClick={() => approveMut.mutate(w.id)}>
                            {t('Approve')}
                          </Button>
                          <Button
                            size="sm"
                            variant="destructive"
                            onClick={() => setRejectFor(w.id)}
                          >
                            <X className="size-4" /> {t('Reject')}
                          </Button>
                        </>
                      )}
                      {w.status === WITHDRAW_STATUS.APPROVED && (
                        <Button size="sm" onClick={() => setPayFor(w.id)}>
                          <Wallet className="size-4" /> {t('Mark Paid')}
                        </Button>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={wfPage}
              pageCount={wfPageCount}
              onChange={setWfPage}
              total={withdraws?.total ?? 0}
            />
          </TabsContent>

          <TabsContent value="sales-users" className="mt-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>{t('Username')}</TableHead>
                  <TableHead>{t('Display')}</TableHead>
                  <TableHead className="text-right">{t('Fixed Rate')}</TableHead>
                  <TableHead className="text-right">{t('Balance')}</TableHead>
                  <TableHead className="text-right">{t('History')}</TableHead>
                  <TableHead />
                </TableRow>
              </TableHeader>
              <TableBody>
                {salesUsers?.items?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground">
                      {t('No sales users yet. Promote a regular user via the Users page.')}
                    </TableCell>
                  </TableRow>
                )}
                {salesUsers?.items?.map((u) => (
                  <TableRow key={u.id}>
                    <TableCell>{u.id}</TableCell>
                    <TableCell>{u.username}</TableCell>
                    <TableCell>{u.display_name ?? '-'}</TableCell>
                    <TableCell className="text-right">
                      {u.commission_rate > 0 ? formatBP(u.commission_rate) : t('Tier-based')}
                    </TableCell>
                    <TableCell className="text-right">{formatUSD(u.commission_balance)}</TableCell>
                    <TableCell className="text-right">{formatUSD(u.commission_history_total)}</TableCell>
                    <TableCell>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() =>
                          openEditTier(u.id, u.commission_rate, u.commission_tier_config)
                        }
                      >
                        <Settings2 className="size-4" /> {t('Tiers')}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={salesPage}
              pageCount={salesPageCount}
              onChange={setSalesPage}
              total={salesUsers?.total ?? 0}
            />
          </TabsContent>

          <TabsContent value="default-tiers" className="mt-4">
            <Card>
              <CardHeader>
                <CardTitle className="text-base">{t('Global default tier configuration')}</CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('From (quota)')}</TableHead>
                      <TableHead>{t('From (USD)')}</TableHead>
                      <TableHead className="text-right">{t('Rate')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {defaultTiers.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={3} className="text-center text-muted-foreground">
                          {t('No tier configured')}
                        </TableCell>
                      </TableRow>
                    )}
                    {defaultTiers.map((tier, idx) => (
                      <TableRow key={idx}>
                        <TableCell>{tier.from}</TableCell>
                        <TableCell>{formatUSD(tier.from)}</TableCell>
                        <TableCell className="text-right">{formatBP(tier.rate_bp)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                <p className="text-xs text-muted-foreground mt-3">
                  {t('Edit the JSON below; rate_bp is in basis points (1000 = 10%), capped at 2000 (20%).')}
                </p>
                <Textarea
                  rows={8}
                  className="font-mono text-xs mt-2"
                  placeholder={JSON.stringify(defaultTiers, null, 2)}
                  value={defaultJSON}
                  onChange={(e) => setDefaultJSON(e.target.value)}
                />
                <div className="mt-3 flex gap-2">
                  <Button
                    variant="outline"
                    onClick={() => setDefaultJSON(JSON.stringify(defaultTiers, null, 2))}
                  >
                    {t('Load current')}
                  </Button>
                  <Button onClick={submitDefaults} disabled={setDefaultsMut.isPending}>
                    {t('Save Defaults')}
                  </Button>
                </div>
              </CardContent>
            </Card>
            <p className="text-xs text-muted-foreground mt-3">
              {t('Hint: 1 USD = {{n}} quota.', { n: QUOTA_PER_USD })}
            </p>
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>

      <Dialog open={generateOpen} onOpenChange={setGenerateOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Generate Monthly Bills')}</DialogTitle>
            <DialogDescription>
              {t('For each sales user, compute monthly bill in draft state. Already-existing bills are skipped.')}
            </DialogDescription>
          </DialogHeader>
          <div>
            <Label>{t('Month')}</Label>
            <Input
              type="month"
              value={generateMonth}
              onChange={(e) => setGenerateMonth(e.target.value)}
            />
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setGenerateOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button onClick={() => generateMut.mutate(generateMonth)} disabled={generateMut.isPending}>
              {t('Generate')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={rejectFor !== null} onOpenChange={(o) => !o && setRejectFor(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Reject Withdraw')}</DialogTitle>
            <DialogDescription>
              {t('Held amount will be returned to the sales balance.')}
            </DialogDescription>
          </DialogHeader>
          <Textarea
            rows={3}
            value={rejectReason}
            onChange={(e) => setRejectReason(e.target.value)}
            placeholder={t('Reason')}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setRejectFor(null)}>
              {t('Cancel')}
            </Button>
            <Button
              variant="destructive"
              onClick={() =>
                rejectFor !== null && rejectMut.mutate({ id: rejectFor, reason: rejectReason })
              }
            >
              {t('Reject')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={payFor !== null} onOpenChange={(o) => !o && setPayFor(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Mark Paid')}</DialogTitle>
            <DialogDescription>
              {t('Record the payout reference (bank transfer ID, etc.).')}
            </DialogDescription>
          </DialogHeader>
          <Input
            value={payNote}
            onChange={(e) => setPayNote(e.target.value)}
            placeholder={t('Payout reference / note')}
          />
          <DialogFooter>
            <Button variant="outline" onClick={() => setPayFor(null)}>
              {t('Cancel')}
            </Button>
            <Button onClick={() => payFor !== null && paidMut.mutate({ id: payFor, note: payNote })}>
              {t('Mark Paid')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={editTierFor !== null} onOpenChange={(o) => !o && setEditTierFor(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Edit Sales Tier Config')}</DialogTitle>
            <DialogDescription>
              {t('Fixed rate or tier JSON. Fixed > 0 takes precedence over JSON. Empty JSON falls back to global defaults.')}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label>{t('Fixed Rate (basis points, 0 = use tiers)')}</Label>
              <Input
                type="number"
                value={editTierRate}
                onChange={(e) => setEditTierRate(e.target.value)}
                min={0}
                max={2000}
              />
            </div>
            <div>
              <Label>{t('Tier JSON (Array of {from, rate_bp})')}</Label>
              <Textarea
                rows={6}
                className="font-mono text-xs"
                value={editTierJSON}
                onChange={(e) => setEditTierJSON(e.target.value)}
                placeholder='[{"from":0,"rate_bp":1000}]'
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setEditTierFor(null)}>
              {t('Cancel')}
            </Button>
            <Button onClick={submitEditTier} disabled={updateTierMut.isPending}>
              {t('Save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </SectionPageLayout>
  )
}

function Pagination({
  page,
  pageCount,
  onChange,
  total,
}: {
  page: number
  pageCount: number
  onChange: (p: number) => void
  total: number
}) {
  const { t } = useTranslation()
  if (total === 0) return null
  return (
    <div className="flex items-center justify-between mt-3 text-sm">
      <span className="text-muted-foreground">
        {t('Total')}: {total}
      </span>
      <div className="flex items-center gap-2">
        <Button
          size="sm"
          variant="outline"
          onClick={() => onChange(Math.max(1, page - 1))}
          disabled={page <= 1}
        >
          {t('Prev')}
        </Button>
        <span>
          {page} / {pageCount}
        </span>
        <Button
          size="sm"
          variant="outline"
          onClick={() => onChange(Math.min(pageCount, page + 1))}
          disabled={page >= pageCount}
        >
          {t('Next')}
        </Button>
      </div>
    </div>
  )
}
