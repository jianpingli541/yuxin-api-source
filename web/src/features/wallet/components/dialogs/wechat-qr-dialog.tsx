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
import { useEffect, useRef, useState } from 'react'
import { QRCodeCanvas } from 'qrcode.react'
import { useTranslation } from 'react-i18next'
import { CheckCircle2, Loader2, XCircle } from 'lucide-react'

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { queryWechatNativeOrder } from '../../api'

interface WechatQrDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  codeUrl: string | null
  orderId: string | null
  expireAt: number
  topupAmount: number
  onSuccess: () => void
}

const POLL_INTERVAL_MS = 3000
const POLL_MAX_ATTEMPTS = 600 // 30 min @ 3s

export function WechatQrDialog({
  open,
  onOpenChange,
  codeUrl,
  orderId,
  expireAt,
  topupAmount,
  onSuccess,
}: WechatQrDialogProps) {
  const { t } = useTranslation()
  const [status, setStatus] = useState<'pending' | 'success' | 'expired' | 'failed'>(
    'pending'
  )
  const attemptsRef = useRef(0)
  const onSuccessRef = useRef(onSuccess)
  onSuccessRef.current = onSuccess

  // 轮询订单状态
  useEffect(() => {
    if (!open || !orderId) return
    setStatus('pending')
    attemptsRef.current = 0

    let cancelled = false

    const poll = async () => {
      while (!cancelled && attemptsRef.current < POLL_MAX_ATTEMPTS) {
        attemptsRef.current += 1
        try {
          const res = await queryWechatNativeOrder(orderId)
          const status = res?.data?.status
          if (status === 'paid') {
            if (!cancelled) setStatus('success')
            setTimeout(() => {
              if (!cancelled) onSuccessRef.current()
            }, 1500)
            return
          }
          if (status === 'expired') {
            if (!cancelled) setStatus('expired')
            return
          }
          if (status === 'failed') {
            if (!cancelled) setStatus('failed')
            return
          }
        } catch {
          // 单次查询失败不中断轮询
        }
        await new Promise<void>((resolve) => setTimeout(resolve, POLL_INTERVAL_MS))
      }
      if (!cancelled) setStatus('expired')
    }

    void poll()
    return () => {
      cancelled = true
    }
  }, [open, orderId])

  // 过期检测
  useEffect(() => {
    if (!open || !expireAt) return
    const ms = expireAt * 1000 - Date.now()
    if (ms <= 0) {
      setStatus('expired')
      return
    }
    const timer = setTimeout(() => setStatus('expired'), ms)
    return () => clearTimeout(timer)
  }, [open, expireAt])

  const handleClose = () => {
    onOpenChange(false)
  }

  const renderBody = () => {
    if (status === 'success') {
      return (
        <div className='flex flex-col items-center gap-3 py-8'>
          <CheckCircle2 className='h-16 w-16 text-green-500' />
          <p className='text-lg font-medium'>{t('Payment Successful')}</p>
          <p className='text-muted-foreground text-sm'>
            {t('Refreshing balance...')}
          </p>
        </div>
      )
    }
    if (status === 'expired') {
      return (
        <div className='flex flex-col items-center gap-3 py-8'>
          <XCircle className='h-16 w-16 text-red-500' />
          <p className='text-lg font-medium'>{t('QR Code Expired')}</p>
          <p className='text-muted-foreground text-sm'>
            {t('Please close and try again.')}
          </p>
        </div>
      )
    }
    return (
      <>
        {codeUrl ? (
          <div className='rounded-lg border bg-white p-4'>
            <QRCodeCanvas
              value={codeUrl}
              size={220}
              includeMargin
              level='M'
            />
          </div>
        ) : (
          <Loader2 className='text-muted-foreground h-8 w-8 animate-spin' />
        )}
        <p className='text-muted-foreground text-center text-sm'>
          {t('Amount')}: ¥{topupAmount.toFixed(2)}
        </p>
        <p className='text-muted-foreground flex items-center gap-1.5 text-xs'>
          <Loader2 className='h-3 w-3 animate-spin' />
          {t('Waiting for payment...')}
        </p>
      </>
    )
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-sm'>
        <DialogHeader>
          <DialogTitle className='text-xl font-semibold'>
            {t('WeChat Pay')}
          </DialogTitle>
          <DialogDescription>
            {t('Scan the QR code with WeChat to pay')}
          </DialogDescription>
        </DialogHeader>

        <div className='flex flex-col items-center gap-4 py-2'>
          {renderBody()}
        </div>

        <div className='flex justify-end'>
          <Button variant='outline' onClick={handleClose}>
            {t('Close')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
