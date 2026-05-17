import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Download, Edit, Plus, RefreshCcw, Upload } from 'lucide-react'
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
  SelectGroup,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { SectionPageLayout } from '@/components/layout'
import {
  createSensitiveRule,
  exportSensitiveHits,
  getSensitiveHits,
  getSensitiveRules,
  markSensitiveHitFalsePositive,
  reloadSensitiveRules,
  seedDefaultSensitiveRules,
  updateSensitiveRule,
} from './api'
import {
  SENSITIVE_RULE_ACTION,
  type ApiResponse,
  type SensitiveHit,
  type SensitiveHitFilters,
  type SensitiveRule,
  type SensitiveRuleFormData,
} from './types'

const PAGE_SIZE = 50

const DEFAULT_FORM: SensitiveRuleFormData = {
  pattern: '',
  is_regex: false,
  enabled: true,
  action: SENSITIVE_RULE_ACTION.MONITOR,
  category: '',
  severity: 3,
  description: '',
}

const DEFAULT_HIT_FILTERS: SensitiveHitFilters = {
  action: undefined,
  keyword: '',
  username: '',
  token_name: '',
  model_name: '',
  request_id: '',
  path: '',
  false_positive: '',
  start_time: '',
  end_time: '',
}

function formatDate(value?: string | null) {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

function normalizeFilterText(value?: string) {
  const normalized = value?.trim()
  return normalized ? normalized : undefined
}

function normalizeFilterTime(value?: string) {
  if (!value) return undefined
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return undefined
  return date.toISOString()
}

function normalizeHitFilters(filters: SensitiveHitFilters) {
  return {
    action: filters.action || undefined,
    keyword: normalizeFilterText(filters.keyword),
    username: normalizeFilterText(filters.username),
    token_name: normalizeFilterText(filters.token_name),
    model_name: normalizeFilterText(filters.model_name),
    request_id: normalizeFilterText(filters.request_id),
    path: normalizeFilterText(filters.path),
    false_positive: filters.false_positive || undefined,
    start_time: normalizeFilterTime(filters.start_time),
    end_time: normalizeFilterTime(filters.end_time),
  }
}

function downloadBlob(blob: Blob, filename: string) {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

function useActionLabel() {
  const { t } = useTranslation()
  return (action: number) =>
    action === SENSITIVE_RULE_ACTION.BLOCK ? t('Block') : t('Monitor')
}

type HitsFilterBarProps = {
  filters: SensitiveHitFilters
  onChange: (filters: SensitiveHitFilters) => void
  onApply: () => void
  onReset: () => void
  onExport: () => void
  exporting: boolean
}

function HitsFilterBar(props: HitsFilterBarProps) {
  const { t } = useTranslation()

  return (
    <form
      className='grid gap-3 rounded-lg border p-3 md:grid-cols-4 xl:grid-cols-5'
      onSubmit={(event) => {
        event.preventDefault()
        props.onApply()
      }}
    >
      <div className='grid gap-1.5'>
        <Label>{t('Action')}</Label>
        <Select
          items={[
            { value: '0', label: t('All') },
            { value: '2', label: t('Monitor') },
            { value: '1', label: t('Block') },
          ]}
          value={String(props.filters.action ?? 0)}
          onValueChange={(value) =>
            props.onChange({
              ...props.filters,
              action: value === '0' ? undefined : Number(value),
            })
          }
        >
          <SelectTrigger className='w-full'>
            <SelectValue placeholder={t('All')} />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              <SelectItem value='0'>{t('All')}</SelectItem>
              <SelectItem value='2'>{t('Monitor')}</SelectItem>
              <SelectItem value='1'>{t('Block')}</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      <div className='grid gap-1.5'>
        <Label htmlFor='sensitive-hit-keyword'>{t('Keyword')}</Label>
        <Input
          id='sensitive-hit-keyword'
          value={props.filters.keyword ?? ''}
          onChange={(event) =>
            props.onChange({ ...props.filters, keyword: event.target.value })
          }
        />
      </div>

      <div className='grid gap-1.5'>
        <Label>{t('False positive')}</Label>
        <Select
          items={[
            { value: 'all', label: t('All') },
            { value: 'false', label: t('Only valid') },
            { value: 'true', label: t('Only false positives') },
          ]}
          value={props.filters.false_positive || 'all'}
          onValueChange={(value) =>
            props.onChange({
              ...props.filters,
              false_positive:
                value === 'true' || value === 'false' ? value : '',
            })
          }
        >
          <SelectTrigger className='w-full'>
            <SelectValue placeholder={t('All')} />
          </SelectTrigger>
          <SelectContent alignItemWithTrigger={false}>
            <SelectGroup>
              <SelectItem value='all'>{t('All')}</SelectItem>
              <SelectItem value='false'>{t('Only valid')}</SelectItem>
              <SelectItem value='true'>{t('Only false positives')}</SelectItem>
            </SelectGroup>
          </SelectContent>
        </Select>
      </div>

      <div className='grid gap-1.5'>
        <Label htmlFor='sensitive-hit-username'>{t('Username')}</Label>
        <Input
          id='sensitive-hit-username'
          value={props.filters.username ?? ''}
          onChange={(event) =>
            props.onChange({ ...props.filters, username: event.target.value })
          }
        />
      </div>

      <div className='grid gap-1.5'>
        <Label htmlFor='sensitive-hit-token'>{t('Token Name')}</Label>
        <Input
          id='sensitive-hit-token'
          value={props.filters.token_name ?? ''}
          onChange={(event) =>
            props.onChange({ ...props.filters, token_name: event.target.value })
          }
        />
      </div>

      <div className='grid gap-1.5'>
        <Label htmlFor='sensitive-hit-model'>{t('Model Name')}</Label>
        <Input
          id='sensitive-hit-model'
          value={props.filters.model_name ?? ''}
          onChange={(event) =>
            props.onChange({ ...props.filters, model_name: event.target.value })
          }
        />
      </div>

      <div className='grid gap-1.5'>
        <Label htmlFor='sensitive-hit-request-id'>{t('Request ID')}</Label>
        <Input
          id='sensitive-hit-request-id'
          value={props.filters.request_id ?? ''}
          onChange={(event) =>
            props.onChange({ ...props.filters, request_id: event.target.value })
          }
        />
      </div>

      <div className='grid gap-1.5'>
        <Label htmlFor='sensitive-hit-path'>{t('Path')}</Label>
        <Input
          id='sensitive-hit-path'
          value={props.filters.path ?? ''}
          onChange={(event) =>
            props.onChange({ ...props.filters, path: event.target.value })
          }
        />
      </div>

      <div className='grid gap-1.5'>
        <Label htmlFor='sensitive-hit-start-time'>{t('Start Time')}</Label>
        <Input
          id='sensitive-hit-start-time'
          type='datetime-local'
          value={props.filters.start_time ?? ''}
          onChange={(event) =>
            props.onChange({ ...props.filters, start_time: event.target.value })
          }
        />
      </div>

      <div className='grid gap-1.5'>
        <Label htmlFor='sensitive-hit-end-time'>{t('End Time')}</Label>
        <Input
          id='sensitive-hit-end-time'
          type='datetime-local'
          value={props.filters.end_time ?? ''}
          onChange={(event) =>
            props.onChange({ ...props.filters, end_time: event.target.value })
          }
        />
      </div>

      <div className='flex items-end gap-2 md:col-span-4 xl:col-span-1'>
        <Button type='submit' variant='secondary' className='flex-1'>
          {t('Search')}
        </Button>
        <Button type='button' variant='outline' onClick={props.onReset}>
          {t('Clear')}
        </Button>
        <Button
          type='button'
          variant='outline'
          onClick={props.onExport}
          disabled={props.exporting}
        >
          <Download className='size-4' />
          {t('Download')}
        </Button>
      </div>
    </form>
  )
}

function ActionBadge({ action }: { action: number }) {
  const getActionLabel = useActionLabel()
  return (
    <Badge
      variant={
        action === SENSITIVE_RULE_ACTION.BLOCK ? 'destructive' : 'secondary'
      }
    >
      {getActionLabel(action)}
    </Badge>
  )
}

function RuleDialog({
  open,
  rule,
  onOpenChange,
  onSubmit,
  submitting,
}: {
  open: boolean
  rule: SensitiveRule | null
  onOpenChange: (open: boolean) => void
  onSubmit: (data: SensitiveRuleFormData) => void
  submitting: boolean
}) {
  const { t } = useTranslation()
  const [form, setForm] = useState<SensitiveRuleFormData>(DEFAULT_FORM)

  useEffect(() => {
    if (!open) return
    setForm(
      rule
        ? {
            pattern: rule.pattern,
            is_regex: rule.is_regex,
            enabled: rule.enabled,
            action: rule.action,
            category: rule.category || '',
            severity: rule.severity || 3,
            description: rule.description || '',
          }
        : DEFAULT_FORM
    )
  }, [open, rule])

  const canSubmit = form.pattern.trim().length > 0 && !submitting

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='sm:max-w-xl'>
        <DialogHeader>
          <DialogTitle>
            {rule ? t('Edit sensitive rule') : t('Create sensitive rule')}
          </DialogTitle>
          <DialogDescription>
            {t('Sensitive rules run asynchronously after relay requests.')}
          </DialogDescription>
        </DialogHeader>

        <div className='grid gap-4'>
          <div className='grid gap-2'>
            <Label htmlFor='sensitive-rule-pattern'>{t('Pattern')}</Label>
            <Input
              id='sensitive-rule-pattern'
              value={form.pattern}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  pattern: event.target.value,
                }))
              }
              placeholder={t('Enter sensitive pattern')}
            />
          </div>

          <div className='grid gap-2 sm:grid-cols-2'>
            <div className='grid gap-2'>
              <Label>{t('Action')}</Label>
              <Select
                items={[
                  { value: '2', label: t('Monitor') },
                  { value: '1', label: t('Block') },
                ]}
                value={String(form.action)}
                onValueChange={(value) =>
                  setForm((current) => ({
                    ...current,
                    action:
                      value === '1'
                        ? SENSITIVE_RULE_ACTION.BLOCK
                        : SENSITIVE_RULE_ACTION.MONITOR,
                  }))
                }
              >
                <SelectTrigger className='w-full'>
                  <SelectValue placeholder={t('Select action')} />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    <SelectItem value='2'>{t('Monitor')}</SelectItem>
                    <SelectItem value='1'>{t('Block')}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>

            <div className='flex items-end justify-between gap-4 rounded-lg border px-3 py-2.5'>
              <div>
                <div className='text-sm font-medium'>{t('Enabled')}</div>
                <div className='text-muted-foreground text-xs'>
                  {t('Use this rule for new relay requests')}
                </div>
              </div>
              <Switch
                checked={form.enabled}
                onCheckedChange={(enabled) =>
                  setForm((current) => ({ ...current, enabled }))
                }
              />
            </div>
          </div>

          <div className='flex items-center justify-between gap-4 rounded-lg border px-3 py-2.5'>
            <div>
              <div className='text-sm font-medium'>
                {t('Regular expression')}
              </div>
              <div className='text-muted-foreground text-xs'>
                {t('Validate the pattern as a Go regular expression')}
              </div>
            </div>
            <Switch
              checked={form.is_regex}
              onCheckedChange={(is_regex) =>
                setForm((current) => ({ ...current, is_regex }))
              }
            />
          </div>

          <div className='grid gap-2 sm:grid-cols-2'>
            <div className='grid gap-2'>
              <Label htmlFor='sensitive-rule-category'>{t('Category')}</Label>
              <Input
                id='sensitive-rule-category'
                value={form.category}
                onChange={(event) =>
                  setForm((current) => ({
                    ...current,
                    category: event.target.value,
                  }))
                }
                placeholder={t('e.g. sexual_assault')}
              />
            </div>
            <div className='grid gap-2'>
              <Label>{t('Severity')}</Label>
              <Select
                items={[
                  { value: '1', label: t('Low (1)') },
                  { value: '2', label: t('Medium-low (2)') },
                  { value: '3', label: t('Medium (3)') },
                  { value: '4', label: t('High (4)') },
                  { value: '5', label: t('Critical (5)') },
                ]}
                value={String(form.severity)}
                onValueChange={(value) =>
                  setForm((current) => ({
                    ...current,
                    severity: Number(value) || 3,
                  }))
                }
              >
                <SelectTrigger className='w-full'>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent alignItemWithTrigger={false}>
                  <SelectGroup>
                    <SelectItem value='1'>{t('Low (1)')}</SelectItem>
                    <SelectItem value='2'>{t('Medium-low (2)')}</SelectItem>
                    <SelectItem value='3'>{t('Medium (3)')}</SelectItem>
                    <SelectItem value='4'>{t('High (4)')}</SelectItem>
                    <SelectItem value='5'>{t('Critical (5)')}</SelectItem>
                  </SelectGroup>
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className='grid gap-2'>
            <Label htmlFor='sensitive-rule-description'>
              {t('Description')}
            </Label>
            <Textarea
              id='sensitive-rule-description'
              value={form.description}
              onChange={(event) =>
                setForm((current) => ({
                  ...current,
                  description: event.target.value,
                }))
              }
              placeholder={t('Describe this rule')}
            />
          </div>
        </div>

        <DialogFooter>
          <Button
            variant='outline'
            type='button'
            onClick={() => onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button
            type='button'
            disabled={!canSubmit}
            onClick={() => onSubmit({ ...form, pattern: form.pattern.trim() })}
          >
            {submitting ? t('Saving...') : t('Save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function RulesTable({
  rules,
  isLoading,
  onEdit,
}: {
  rules: SensitiveRule[]
  isLoading: boolean
  onEdit: (rule: SensitiveRule) => void
}) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <div className='text-muted-foreground py-10 text-sm'>
        {t('Loading...')}
      </div>
    )
  }

  if (rules.length === 0) {
    return (
      <div className='text-muted-foreground py-10 text-sm'>
        {t('No sensitive rules found')}
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('ID')}</TableHead>
          <TableHead>{t('Pattern')}</TableHead>
          <TableHead>{t('Type')}</TableHead>
          <TableHead>{t('Action')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          <TableHead>{t('Hits')}</TableHead>
          <TableHead>{t('Last Hit')}</TableHead>
          <TableHead>{t('Description')}</TableHead>
          <TableHead className='text-right'>{t('Actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {rules.map((rule) => (
          <TableRow key={rule.id}>
            <TableCell>{rule.id}</TableCell>
            <TableCell className='max-w-[280px] truncate font-mono text-xs'>
              {rule.pattern}
            </TableCell>
            <TableCell>
              <Badge variant='outline'>
                {rule.is_regex ? t('Regex') : t('Plain text')}
              </Badge>
            </TableCell>
            <TableCell>
              <ActionBadge action={rule.action} />
            </TableCell>
            <TableCell>
              <Badge variant={rule.enabled ? 'secondary' : 'outline'}>
                {rule.enabled ? t('Enabled') : t('Disabled')}
              </Badge>
            </TableCell>
            <TableCell>{rule.hit_count}</TableCell>
            <TableCell>{formatDate(rule.last_hit_at)}</TableCell>
            <TableCell className='max-w-[260px] truncate'>
              {rule.description || '-'}
            </TableCell>
            <TableCell className='text-right'>
              <Button
                type='button'
                variant='ghost'
                size='icon-sm'
                aria-label={t('Edit sensitive rule')}
                onClick={() => onEdit(rule)}
              >
                <Edit className='size-4' />
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

function HitsTable({
  hits,
  isLoading,
  onMarkFalsePositive,
  markingHitId,
}: {
  hits: SensitiveHit[]
  isLoading: boolean
  onMarkFalsePositive: (hitId: number, value: boolean) => void
  markingHitId: number | null
}) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <div className='text-muted-foreground py-10 text-sm'>
        {t('Loading...')}
      </div>
    )
  }

  if (hits.length === 0) {
    return (
      <div className='text-muted-foreground py-10 text-sm'>
        {t('No sensitive hits found')}
      </div>
    )
  }

  return (
    <Table>
      <TableHeader>
        <TableRow>
          <TableHead>{t('Time')}</TableHead>
          <TableHead>{t('Pattern')}</TableHead>
          <TableHead>{t('Action')}</TableHead>
          <TableHead>{t('User')}</TableHead>
          <TableHead>{t('Token')}</TableHead>
          <TableHead>{t('Model')}</TableHead>
          <TableHead>{t('Request ID')}</TableHead>
          <TableHead>{t('Path')}</TableHead>
          <TableHead>{t('Prompt Snippet')}</TableHead>
          <TableHead>{t('Status')}</TableHead>
          <TableHead className='text-right'>{t('Actions')}</TableHead>
        </TableRow>
      </TableHeader>
      <TableBody>
        {hits.map((hit) => (
          <TableRow key={hit.id}>
            <TableCell>{formatDate(hit.created_at)}</TableCell>
            <TableCell className='max-w-[220px] truncate font-mono text-xs'>
              {hit.pattern}
            </TableCell>
            <TableCell>
              <ActionBadge action={hit.action} />
            </TableCell>
            <TableCell>{hit.username || hit.user_id || '-'}</TableCell>
            <TableCell>{hit.token_name || hit.token_id || '-'}</TableCell>
            <TableCell>{hit.model_name || '-'}</TableCell>
            <TableCell className='max-w-[180px] truncate font-mono text-xs'>
              {hit.request_id || '-'}
            </TableCell>
            <TableCell className='max-w-[180px] truncate'>{hit.path}</TableCell>
            <TableCell className='max-w-[360px] whitespace-normal'>
              {hit.prompt_snippet}
            </TableCell>
            <TableCell>
              {hit.false_positive ? (
                <Badge variant='outline'>{t('False positive')}</Badge>
              ) : (
                <Badge variant='secondary'>{t('Valid')}</Badge>
              )}
            </TableCell>
            <TableCell className='text-right'>
              <Button
                type='button'
                variant='outline'
                size='sm'
                disabled={markingHitId === hit.id}
                onClick={() => onMarkFalsePositive(hit.id, !hit.false_positive)}
              >
                {hit.false_positive
                  ? t('Unmark false positive')
                  : t('Mark false positive')}
              </Button>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  )
}

export function SensitiveMonitor() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [rulePage, setRulePage] = useState(1)
  const [hitPage, setHitPage] = useState(1)
  const [hitFilterDraft, setHitFilterDraft] =
    useState<SensitiveHitFilters>(DEFAULT_HIT_FILTERS)
  const [hitFilters, setHitFilters] =
    useState<SensitiveHitFilters>(DEFAULT_HIT_FILTERS)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [editingRule, setEditingRule] = useState<SensitiveRule | null>(null)
  const hitQueryParams = normalizeHitFilters(hitFilters)

  const rulesQuery = useQuery({
    queryKey: ['sensitive-monitor', 'rules', rulePage],
    queryFn: () => getSensitiveRules({ p: rulePage, page_size: PAGE_SIZE }),
  })

  const hitsQuery = useQuery({
    queryKey: ['sensitive-monitor', 'hits', hitPage, hitQueryParams],
    queryFn: () =>
      getSensitiveHits({
        p: hitPage,
        page_size: PAGE_SIZE,
        ...hitQueryParams,
      }),
  })

  const rules = rulesQuery.data?.data?.items ?? []
  const hits = hitsQuery.data?.data?.items ?? []
  const rulesTotal = rulesQuery.data?.data?.total ?? 0
  const hitsTotal = hitsQuery.data?.data?.total ?? 0

  const invalidateSensitiveData = () =>
    queryClient.invalidateQueries({ queryKey: ['sensitive-monitor'] })

  const saveRuleMutation = useMutation<
    ApiResponse<SensitiveRule | null>,
    Error,
    SensitiveRuleFormData
  >({
    mutationFn: (data: SensitiveRuleFormData) =>
      editingRule
        ? updateSensitiveRule(editingRule.id, data)
        : createSensitiveRule(data),
    onSuccess: (res) => {
      if (!res.success) return
      toast.success(
        editingRule
          ? t('Sensitive rule updated successfully')
          : t('Sensitive rule created successfully')
      )
      setDialogOpen(false)
      setEditingRule(null)
      invalidateSensitiveData()
    },
  })

  const reloadMutation = useMutation({
    mutationFn: reloadSensitiveRules,
    onSuccess: (res) => {
      if (!res.success) return
      toast.success(t('Sensitive monitor cache reloaded'))
    },
  })

  const seedMutation = useMutation({
    mutationFn: seedDefaultSensitiveRules,
    onSuccess: (res) => {
      if (!res.success) return
      const created = res.data?.created ?? 0
      const updated = res.data?.updated ?? 0
      toast.success(
        t(
          'Default sensitive rules imported: {{created}} created, {{updated}} updated',
          {
            created,
            updated,
          }
        )
      )
      invalidateSensitiveData()
    },
  })

  const exportHitsMutation = useMutation({
    mutationFn: exportSensitiveHits,
    onSuccess: (blob) => {
      downloadBlob(
        blob,
        `sensitive-monitor-hits-${new Date()
          .toISOString()
          .slice(0, 19)
          .replace(/[:T]/g, '-')}.csv`
      )
      toast.success(t('Sensitive hits exported'))
    },
  })

  const markFalsePositiveMutation = useMutation({
    mutationFn: ({ hitId, value }: { hitId: number; value: boolean }) =>
      markSensitiveHitFalsePositive(hitId, value),
    onSuccess: (res, variables) => {
      if (!res.success) return
      toast.success(
        variables.value
          ? t('Hit marked as false positive')
          : t('False positive mark removed')
      )
      invalidateSensitiveData()
    },
  })

  const applyHitFilters = () => {
    setHitPage(1)
    setHitFilters(hitFilterDraft)
  }

  const resetHitFilters = () => {
    setHitPage(1)
    setHitFilterDraft(DEFAULT_HIT_FILTERS)
    setHitFilters(DEFAULT_HIT_FILTERS)
  }

  const exportHitFilters = () => {
    const filters = hitFilterDraft
    setHitPage(1)
    setHitFilters(filters)
    exportHitsMutation.mutate(normalizeHitFilters(filters))
  }

  const rulePageCount = Math.max(1, Math.ceil(rulesTotal / PAGE_SIZE))
  const hitPageCount = Math.max(1, Math.ceil(hitsTotal / PAGE_SIZE))

  return (
    <SectionPageLayout>
      <SectionPageLayout.Title>
        {t('Sensitive Monitor')}
      </SectionPageLayout.Title>
      <SectionPageLayout.Actions>
        <Button
          type='button'
          variant='outline'
          onClick={() => seedMutation.mutate()}
          disabled={seedMutation.isPending}
        >
          <Upload className='size-4' />
          {t('Seed default rules')}
        </Button>
        <Button
          type='button'
          variant='outline'
          onClick={() => reloadMutation.mutate()}
          disabled={reloadMutation.isPending}
        >
          <RefreshCcw className='size-4' />
          {t('Reload cache')}
        </Button>
        <Button
          type='button'
          onClick={() => {
            setEditingRule(null)
            setDialogOpen(true)
          }}
        >
          <Plus className='size-4' />
          {t('Create rule')}
        </Button>
      </SectionPageLayout.Actions>
      <SectionPageLayout.Content>
        <Tabs defaultValue='rules' className='space-y-4'>
          <TabsList>
            <TabsTrigger value='rules'>{t('Rules')}</TabsTrigger>
            <TabsTrigger value='hits'>{t('Hits')}</TabsTrigger>
          </TabsList>

          <TabsContent value='rules' className='space-y-3'>
            <RulesTable
              rules={rules}
              isLoading={rulesQuery.isLoading}
              onEdit={(rule) => {
                setEditingRule(rule)
                setDialogOpen(true)
              }}
            />
            <PaginationBar
              page={rulePage}
              pageCount={rulePageCount}
              total={rulesTotal}
              onPageChange={setRulePage}
            />
          </TabsContent>

          <TabsContent value='hits' className='space-y-3'>
            <HitsFilterBar
              filters={hitFilterDraft}
              onChange={setHitFilterDraft}
              onApply={applyHitFilters}
              onReset={resetHitFilters}
              onExport={exportHitFilters}
              exporting={exportHitsMutation.isPending}
            />
            <HitsTable
              hits={hits}
              isLoading={hitsQuery.isLoading}
              markingHitId={
                markFalsePositiveMutation.isPending
                  ? (markFalsePositiveMutation.variables?.hitId ?? null)
                  : null
              }
              onMarkFalsePositive={(hitId, value) =>
                markFalsePositiveMutation.mutate({ hitId, value })
              }
            />
            <PaginationBar
              page={hitPage}
              pageCount={hitPageCount}
              total={hitsTotal}
              onPageChange={setHitPage}
            />
          </TabsContent>
        </Tabs>
      </SectionPageLayout.Content>

      <RuleDialog
        open={dialogOpen}
        rule={editingRule}
        submitting={saveRuleMutation.isPending}
        onOpenChange={(open) => {
          setDialogOpen(open)
          if (!open) setEditingRule(null)
        }}
        onSubmit={(data) => saveRuleMutation.mutate(data)}
      />
    </SectionPageLayout>
  )
}

function PaginationBar({
  page,
  pageCount,
  total,
  onPageChange,
}: {
  page: number
  pageCount: number
  total: number
  onPageChange: (page: number) => void
}) {
  const { t } = useTranslation()

  return (
    <div className='flex flex-wrap items-center justify-between gap-3 border-t pt-3 text-sm'>
      <div className='text-muted-foreground'>
        {t('{{total}} records', { total })}
      </div>
      <div className='flex items-center gap-2'>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={page <= 1}
          onClick={() => onPageChange(page - 1)}
        >
          {t('Previous')}
        </Button>
        <span className='text-muted-foreground min-w-20 text-center'>
          {t('{{page}} / {{pageCount}}', { page, pageCount })}
        </span>
        <Button
          type='button'
          variant='outline'
          size='sm'
          disabled={page >= pageCount}
          onClick={() => onPageChange(page + 1)}
        >
          {t('Next')}
        </Button>
      </div>
    </div>
  )
}
