import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { RefreshCcw, Send, TrendingUp, UserPlus, Users } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { CopyButton } from '@/components/copy-button'
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
  applyWithdraw,
  getSalesConsumes,
  getSalesDashboard,
  listSalesBills,
  listSalesCustomers,
  listSalesWithdraws,
} from './api'
import {
  BILL_STATUS,
  formatBP,
  formatTs,
  formatUSD,
  QUOTA_PER_USD,
  WITHDRAW_STATUS,
  type CommissionBill,
  type WithdrawRequest,
} from './types'

const PAGE_SIZE = 20

function currentMonth() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}`
}

function billStatusBadge(status: number) {
  if (status === BILL_STATUS.CONFIRMED)
    return <Badge>Confirmed</Badge>
  return <Badge variant="secondary">Draft</Badge>
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

export function Sales() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [tab, setTab] = useState<'dashboard' | 'customers' | 'consumes' | 'bills' | 'withdraws'>('dashboard')
  const [month, setMonth] = useState<string>(currentMonth())
  const [withdrawOpen, setWithdrawOpen] = useState(false)
  const [withdrawAmountUSD, setWithdrawAmountUSD] = useState<string>('')
  const [withdrawNote, setWithdrawNote] = useState('')
  const [customerPage, setCustomerPage] = useState(1)
  const [billsPage, setBillsPage] = useState(1)
  const [withdrawsPage, setWithdrawsPage] = useState(1)

  const dashboardQuery = useQuery({
    queryKey: ['sales', 'dashboard', month],
    queryFn: () => getSalesDashboard(month),
  })
  const consumesQuery = useQuery({
    queryKey: ['sales', 'consumes', month],
    queryFn: () => getSalesConsumes(month),
    enabled: tab === 'consumes',
  })
  const customersQuery = useQuery({
    queryKey: ['sales', 'customers', customerPage],
    queryFn: () => listSalesCustomers({ page: customerPage, page_size: PAGE_SIZE }),
    enabled: tab === 'customers',
  })
  const billsQuery = useQuery({
    queryKey: ['sales', 'bills', billsPage],
    queryFn: () => listSalesBills({ page: billsPage, page_size: PAGE_SIZE }),
    enabled: tab === 'bills',
  })
  const withdrawsQuery = useQuery({
    queryKey: ['sales', 'withdraws', withdrawsPage],
    queryFn: () => listSalesWithdraws({ page: withdrawsPage, page_size: PAGE_SIZE }),
    enabled: tab === 'withdraws',
  })

  const refreshAll = () => queryClient.invalidateQueries({ queryKey: ['sales'] })

  const withdrawMutation = useMutation({
    mutationFn: applyWithdraw,
    onSuccess: () => {
      toast.success(t('Withdraw request submitted'))
      setWithdrawOpen(false)
      setWithdrawAmountUSD('')
      setWithdrawNote('')
      queryClient.invalidateQueries({ queryKey: ['sales'] })
    },
    onError: (err: unknown) => {
      const msg = err instanceof Error ? err.message : t('Submit failed')
      toast.error(msg)
    },
  })

  const submitWithdraw = () => {
    const usd = Number(withdrawAmountUSD)
    if (!Number.isFinite(usd) || usd <= 0) {
      toast.error(t('Please enter a positive amount'))
      return
    }
    const amountQuota = Math.floor(usd * QUOTA_PER_USD)
    withdrawMutation.mutate({ amount_quota: amountQuota, applicant_note: withdrawNote })
  }

  const dashboard = dashboardQuery.data?.data
  const tiers = dashboard?.tiers ?? []
  const balance = dashboard?.commission_balance ?? 0
  const monthCommission = dashboard?.month_commission_quota ?? 0
  const monthConsume = dashboard?.month_consume_quota ?? 0
  const historyTotal = dashboard?.commission_history_total ?? 0
  const customerCount = dashboard?.customer_count ?? 0
  const affCode = dashboard?.aff_code ?? ''
  const affCount = dashboard?.aff_count ?? 0
  const inviteLink = useMemo(() => {
    if (!affCode || typeof window === 'undefined') return ''
    return `${window.location.origin}/sign-up?aff=${affCode}`
  }, [affCode])
  const consumes = consumesQuery.data?.data
  const customers = customersQuery.data?.data
  const bills = billsQuery.data?.data
  const withdraws = withdrawsQuery.data?.data

  const customerPageCount = useMemo(
    () => Math.max(1, Math.ceil((customers?.total ?? 0) / PAGE_SIZE)),
    [customers?.total],
  )
  const billsPageCount = useMemo(
    () => Math.max(1, Math.ceil((bills?.total ?? 0) / PAGE_SIZE)),
    [bills?.total],
  )
  const withdrawsPageCount = useMemo(
    () => Math.max(1, Math.ceil((withdraws?.total ?? 0) / PAGE_SIZE)),
    [withdraws?.total],
  )

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>{t('Sales Center')}</SectionPageLayout.Title>
      <SectionPageLayout.Description>
        {t('Manage your customers, track commission, and submit withdraw requests.')}
      </SectionPageLayout.Description>
      <SectionPageLayout.Actions>
        <Input
          type="month"
          value={month}
          onChange={(e) => setMonth(e.target.value)}
          className="w-40"
        />
        <Button variant="outline" onClick={refreshAll}>
          <RefreshCcw className="size-4" />
          {t('Refresh')}
        </Button>
        <Button onClick={() => setWithdrawOpen(true)} disabled={balance <= 0}>
          <Send className="size-4" />
          {t('Apply Withdraw')}
        </Button>
      </SectionPageLayout.Actions>

      <SectionPageLayout.Content>
        <Tabs value={tab} onValueChange={(v) => setTab(v as typeof tab)}>
          <TabsList>
            <TabsTrigger value="dashboard">{t('Dashboard')}</TabsTrigger>
            <TabsTrigger value="customers">
              {t('Customers')}
              <Badge variant="secondary" className="ml-2">
                {customerCount}
              </Badge>
            </TabsTrigger>
            <TabsTrigger value="consumes">{t('Consumes')}</TabsTrigger>
            <TabsTrigger value="bills">{t('Bills')}</TabsTrigger>
            <TabsTrigger value="withdraws">{t('Withdraws')}</TabsTrigger>
          </TabsList>

          <TabsContent value="dashboard" className="mt-4">
            <Card className="mb-3">
              <CardHeader className="pb-2">
                <CardTitle className="text-base flex items-center gap-2">
                  <UserPlus className="size-4" />
                  {t('Invite Customers')}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <p className="text-xs text-muted-foreground mb-3">
                  {t('Share this link with prospects. Anyone who signs up via this link is automatically bound to you and you will earn commission from their consume.')}
                </p>
                <div className="flex items-center gap-2 flex-wrap">
                  <div className="flex-1 min-w-[280px] rounded-md border bg-muted/40 px-3 py-2 font-mono text-xs break-all">
                    {inviteLink || t('Loading...')}
                  </div>
                  {inviteLink && (
                    <CopyButton value={inviteLink} variant="outline" size="default">
                      {t('Copy Link')}
                    </CopyButton>
                  )}
                </div>
                <div className="mt-3 flex items-center gap-4 text-sm text-muted-foreground">
                  <span>
                    {t('Affiliate code')}: <code className="font-mono">{affCode || '-'}</code>
                  </span>
                  <span>
                    {t('Invited so far')}: <b className="text-foreground">{affCount}</b>
                  </span>
                </div>
              </CardContent>
            </Card>

            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">
                    {t('Withdrawable Balance')}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{formatUSD(balance)}</div>
                  <p className="text-xs text-muted-foreground mt-1">
                    {t('History total')}: {formatUSD(historyTotal)}
                  </p>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">
                    {t('This month consume')}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{formatUSD(monthConsume)}</div>
                  <p className="text-xs text-muted-foreground mt-1">{month}</p>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">
                    {t('This month commission')}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{formatUSD(monthCommission)}</div>
                  <p className="text-xs text-muted-foreground mt-1">{t('Estimated, pre-confirm')}</p>
                </CardContent>
              </Card>
              <Card>
                <CardHeader className="pb-2">
                  <CardTitle className="text-sm font-medium text-muted-foreground">
                    {t('Customers')}
                  </CardTitle>
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold flex items-center gap-2">
                    <Users className="size-5" />
                    {customerCount}
                  </div>
                </CardContent>
              </Card>
            </div>

            <Card className="mt-4">
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <TrendingUp className="size-4" />
                  {t('Commission Tiers')}
                </CardTitle>
              </CardHeader>
              <CardContent>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>{t('From (monthly consume per customer)')}</TableHead>
                      <TableHead className="text-right">{t('Rate')}</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {tiers.length === 0 && (
                      <TableRow>
                        <TableCell colSpan={2} className="text-center text-muted-foreground">
                          {t('No tier configured')}
                        </TableCell>
                      </TableRow>
                    )}
                    {tiers.map((tier, idx) => (
                      <TableRow key={idx}>
                        <TableCell>{formatUSD(tier.from)}+</TableCell>
                        <TableCell className="text-right">{formatBP(tier.rate_bp)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
                <p className="text-xs text-muted-foreground mt-2">
                  {t('Tiers apply per-customer monthly consume (resets each month).')}
                </p>
              </CardContent>
            </Card>
          </TabsContent>

          <TabsContent value="customers" className="mt-4">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>{t('Username')}</TableHead>
                  <TableHead>{t('Display Name')}</TableHead>
                  <TableHead>{t('Email')}</TableHead>
                  <TableHead className="text-right">{t('Used Quota')}</TableHead>
                  <TableHead className="text-right">{t('Requests')}</TableHead>
                  <TableHead>{t('Created')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {customersQuery.isLoading && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground">
                      {t('Loading...')}
                    </TableCell>
                  </TableRow>
                )}
                {!customersQuery.isLoading && customers?.items?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={7} className="text-center text-muted-foreground">
                      {t('No customers bound to you yet')}
                    </TableCell>
                  </TableRow>
                )}
                {customers?.items?.map((u) => (
                  <TableRow key={u.id}>
                    <TableCell>{u.id}</TableCell>
                    <TableCell>{u.username}</TableCell>
                    <TableCell>{u.display_name ?? '-'}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{u.email ?? '-'}</TableCell>
                    <TableCell className="text-right">{formatUSD(u.used_quota ?? 0)}</TableCell>
                    <TableCell className="text-right">{u.request_count ?? 0}</TableCell>
                    <TableCell>{formatTs(u.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
            <Pagination
              page={customerPage}
              pageCount={customerPageCount}
              onChange={setCustomerPage}
              total={customers?.total ?? 0}
            />
          </TabsContent>

          <TabsContent value="consumes" className="mt-4">
            <div className="mb-3 text-sm text-muted-foreground">
              {t('Consume and commission by customer for')} {consumes?.month ?? month}.
              {t('Total consume')}: <b>{formatUSD(consumes?.total_consume ?? 0)}</b> ·{' '}
              {t('Total commission')}: <b>{formatUSD(consumes?.total_commission ?? 0)}</b>
            </div>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('Customer ID')}</TableHead>
                  <TableHead>{t('Customer Name')}</TableHead>
                  <TableHead className="text-right">{t('Consume')}</TableHead>
                  <TableHead className="text-right">{t('Commission')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {consumesQuery.isLoading && (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center text-muted-foreground">
                      {t('Loading...')}
                    </TableCell>
                  </TableRow>
                )}
                {!consumesQuery.isLoading && consumes?.items?.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center text-muted-foreground">
                      {t('No consume in this month')}
                    </TableCell>
                  </TableRow>
                )}
                {consumes?.items?.map((row) => (
                  <TableRow key={row.customer_id}>
                    <TableCell>{row.customer_id}</TableCell>
                    <TableCell>{row.customer_name || '-'}</TableCell>
                    <TableCell className="text-right">{formatUSD(row.consume_quota)}</TableCell>
                    <TableCell className="text-right">{formatUSD(row.commission_quota)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </TabsContent>

          <TabsContent value="bills" className="mt-4">
            <BillsTable bills={bills?.items ?? []} loading={billsQuery.isLoading} />
            <Pagination
              page={billsPage}
              pageCount={billsPageCount}
              onChange={setBillsPage}
              total={bills?.total ?? 0}
            />
          </TabsContent>

          <TabsContent value="withdraws" className="mt-4">
            <WithdrawsTable rows={withdraws?.items ?? []} loading={withdrawsQuery.isLoading} />
            <Pagination
              page={withdrawsPage}
              pageCount={withdrawsPageCount}
              onChange={setWithdrawsPage}
              total={withdraws?.total ?? 0}
            />
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>

      <Dialog open={withdrawOpen} onOpenChange={setWithdrawOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{t('Apply Withdraw')}</DialogTitle>
            <DialogDescription>
              {t('Withdrawable balance')}: <b>{formatUSD(balance)}</b>.{' '}
              {t('Amount you enter (USD) will be deducted from balance immediately and held in pending state.')}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-3">
            <div>
              <Label>{t('Amount (USD)')}</Label>
              <Input
                type="number"
                step="0.01"
                value={withdrawAmountUSD}
                onChange={(e) => setWithdrawAmountUSD(e.target.value)}
                placeholder="0.00"
              />
            </div>
            <div>
              <Label>{t('Note (optional)')}</Label>
              <Textarea
                rows={3}
                value={withdrawNote}
                onChange={(e) => setWithdrawNote(e.target.value)}
                placeholder={t('Bank info, contact, etc.')}
              />
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setWithdrawOpen(false)}>
              {t('Cancel')}
            </Button>
            <Button onClick={submitWithdraw} disabled={withdrawMutation.isPending}>
              {t('Submit')}
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

function BillsTable({ bills, loading }: { bills: CommissionBill[]; loading: boolean }) {
  const { t } = useTranslation()
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Month')}</TableHead>
          <TableHead className="text-right">{t('Total Consume')}</TableHead>
          <TableHead className="text-right">{t('Commission')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          <TableHead>{t('Created')}</TableHead>
          <TableHead>{t('Confirmed')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {loading && (
          <TableRow>
            <TableCell colSpan={6} className="text-center text-muted-foreground">
              {t('Loading...')}
            </TableCell>
          </TableRow>
        )}
        {!loading && bills.length === 0 && (
          <TableRow>
            <TableCell colSpan={6} className="text-center text-muted-foreground">
              {t('No bills yet')}
            </TableCell>
          </TableRow>
        )}
        {bills.map((b) => (
          <TableRow key={b.id}>
            <TableCell>{b.year_month}</TableCell>
            <TableCell className="text-right">{formatUSD(b.total_consume_quota)}</TableCell>
            <TableCell className="text-right">{formatUSD(b.commission_quota)}</TableCell>
            <TableCell>{billStatusBadge(b.status)}</TableCell>
            <TableCell>{formatTs(b.created_at)}</TableCell>
            <TableCell>{formatTs(b.confirmed_at)}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function WithdrawsTable({ rows, loading }: { rows: WithdrawRequest[]; loading: boolean }) {
  const { t } = useTranslation()
  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>ID</TableHead>
          <TableHead className="text-right">{t('Amount')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          <TableHead>{t('Applied')}</TableHead>
          <TableHead>{t('Reviewed')}</TableHead>
          <TableHead>{t('Paid')}</TableHead>
          <TableHead>{t('Note')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {loading && (
          <TableRow>
            <TableCell colSpan={7} className="text-center text-muted-foreground">
              {t('Loading...')}
            </TableCell>
          </TableRow>
        )}
        {!loading && rows.length === 0 && (
          <TableRow>
            <TableCell colSpan={7} className="text-center text-muted-foreground">
              {t('No withdraw requests yet')}
            </TableCell>
          </TableRow>
        )}
        {rows.map((r) => (
          <TableRow key={r.id}>
            <TableCell>{r.id}</TableCell>
            <TableCell className="text-right">{formatUSD(r.amount_quota)}</TableCell>
            <TableCell>{withdrawStatusBadge(r.status)}</TableCell>
            <TableCell>{formatTs(r.applied_at)}</TableCell>
            <TableCell>{formatTs(r.reviewed_at)}</TableCell>
            <TableCell>{formatTs(r.paid_at)}</TableCell>
            <TableCell className="text-xs text-muted-foreground">
              {r.status === WITHDRAW_STATUS.REJECTED ? r.reject_reason : r.paid_note ?? '-'}
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}
