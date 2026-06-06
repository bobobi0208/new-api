/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { ArrowDown, ArrowUp, RotateCcw } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  NativeSelect,
  NativeSelectOption,
} from '@/components/ui/native-select'
import { getChannelLoadOverview, type ChannelLoadRow } from '../../api'
import { handleRecoverChannelAffinity } from '../../lib/channel-actions'

interface Props {
  open: boolean
  onOpenChange: (open: boolean) => void
}

type SortKey = keyof Pick<
  ChannelLoadRow,
  'inflight' | 'rpm' | 'error_rate' | 'failover_entries' | 'requests'
>

const WINDOW_OPTIONS = [5, 15, 60]

export function ChannelLoadOverviewDialog(props: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [loading, setLoading] = useState(false)
  const [windowMinutes, setWindowMinutes] = useState(5)
  const [rows, setRows] = useState<ChannelLoadRow[]>([])
  const [errorLogEnabled, setErrorLogEnabled] = useState(true)
  const [sortKey, setSortKey] = useState<SortKey>('inflight')
  const [sortDesc, setSortDesc] = useState(true)
  const seqRef = useRef(0)

  const load = useCallback(() => {
    const seq = ++seqRef.current
    setLoading(true)
    getChannelLoadOverview(windowMinutes)
      .then((res) => {
        if (seq !== seqRef.current) return
        if (res.success && res.data) {
          setRows(res.data.channels || [])
          setErrorLogEnabled(res.data.error_log_enabled)
        } else {
          toast.error(res.message || t('Request failed'))
        }
      })
      .catch(() => {
        if (seq !== seqRef.current) return
        toast.error(t('Request failed'))
      })
      .finally(() => {
        if (seq !== seqRef.current) return
        setLoading(false)
      })
  }, [windowMinutes, t])

  useEffect(() => {
    if (!props.open) return
    load()
  }, [props.open, load])

  const sortedRows = useMemo(() => {
    const copy = [...rows]
    copy.sort((a, b) => {
      const diff = (a[sortKey] as number) - (b[sortKey] as number)
      return sortDesc ? -diff : diff
    })
    return copy
  }, [rows, sortKey, sortDesc])

  const onSort = (key: SortKey) => {
    if (key === sortKey) {
      setSortDesc((d) => !d)
    } else {
      setSortKey(key)
      setSortDesc(true)
    }
  }

  const sortIcon = (key: SortKey) =>
    key === sortKey ? (
      sortDesc ? (
        <ArrowDown className='ml-1 inline h-3 w-3' />
      ) : (
        <ArrowUp className='ml-1 inline h-3 w-3' />
      )
    ) : null

  const onRecover = async (id: number) => {
    await handleRecoverChannelAffinity(id, queryClient)
    load()
  }

  const headerBtn = (key: SortKey, label: string) => (
    <button
      type='button'
      className='inline-flex items-center hover:underline'
      onClick={() => onSort(key)}
    >
      {label}
      {sortIcon(key)}
    </button>
  )

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className='sm:max-w-4xl'>
        <DialogHeader>
          <DialogTitle>{t('Channel Load Overview')}</DialogTitle>
        </DialogHeader>

        <div className='flex items-center gap-2'>
          <span className='text-muted-foreground text-sm'>
            {t('Window (minutes)')}
          </span>
          <NativeSelect
            value={String(windowMinutes)}
            onChange={(e) => setWindowMinutes(Number(e.target.value))}
            className='w-28'
          >
            {WINDOW_OPTIONS.map((w) => (
              <NativeSelectOption key={w} value={String(w)}>
                {w}
              </NativeSelectOption>
            ))}
          </NativeSelect>
          <Button variant='outline' size='sm' onClick={load} disabled={loading}>
            {t('Refresh')}
          </Button>
        </div>

        {!errorLogEnabled && (
          <p className='text-muted-foreground text-xs'>
            {t(
              'Error logging is disabled, so error rate is unavailable (shown as -).'
            )}
          </p>
        )}

        <div className='max-h-[60vh] overflow-auto'>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('Channel')}</TableHead>
                <TableHead className='text-right'>
                  {headerBtn('inflight', t('In-flight'))}
                </TableHead>
                <TableHead className='text-right'>
                  {headerBtn('rpm', t('RPM'))}
                </TableHead>
                <TableHead className='text-right'>
                  {headerBtn('error_rate', t('Error rate'))}
                </TableHead>
                <TableHead className='text-right'>
                  {headerBtn('failover_entries', t('Failover entries'))}
                </TableHead>
                <TableHead className='text-right'>{t('Actions')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className='text-muted-foreground py-8 text-center text-sm'
                  >
                    {t('Loading...')}
                  </TableCell>
                </TableRow>
              ) : sortedRows.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className='text-muted-foreground py-8 text-center text-sm'
                  >
                    {t('No data available')}
                  </TableCell>
                </TableRow>
              ) : (
                sortedRows.map((r) => (
                  <TableRow key={r.channel_id}>
                    <TableCell>
                      <span className='font-medium'>{r.channel_name}</span>
                      <span className='text-muted-foreground ml-1 text-xs'>
                        #{r.channel_id}
                      </span>
                    </TableCell>
                    <TableCell className='text-right tabular-nums'>
                      {r.inflight}
                    </TableCell>
                    <TableCell className='text-right tabular-nums'>
                      {r.rpm.toFixed(1)}
                    </TableCell>
                    <TableCell className='text-right tabular-nums'>
                      {errorLogEnabled
                        ? `${(r.error_rate * 100).toFixed(1)}%`
                        : '-'}
                    </TableCell>
                    <TableCell className='text-right tabular-nums'>
                      {r.failover_entries}
                    </TableCell>
                    <TableCell className='text-right'>
                      <Button
                        variant='outline'
                        size='sm'
                        disabled={r.failover_entries === 0}
                        onClick={() => onRecover(r.channel_id)}
                      >
                        <RotateCcw className='h-3 w-3' />
                        {t('Recover')}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </DialogContent>
    </Dialog>
  )
}
