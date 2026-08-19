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
import i18next from 'i18next'
import { useState, useCallback } from 'react'
import { toast } from 'sonner'

import { requestWechatPayment, isApiSuccess } from '../api'

interface WechatPayResult {
  code_url: string
  order_id: string
  expire_at: number
}

function getPayResult(data: unknown): WechatPayResult | null {
  if (!data || typeof data !== 'object') return null
  const d = data as Record<string, unknown>
  if (typeof d.code_url === 'string' && typeof d.order_id === 'string') {
    return {
      code_url: d.code_url,
      order_id: d.order_id,
      expire_at: typeof d.expire_at === 'number' ? d.expire_at : 0,
    }
  }
  return null
}

function getErrorMessage(message: string | undefined, data: unknown): string {
  if (typeof data === 'string' && data.trim()) return data
  return message || i18next.t('Payment request failed')
}

/**
 * Hook for handling WeChat (Native / scan-to-pay) payment processing.
 * Returns the QR code_url to render; caller polls order status.
 */
export function useWechatPayment() {
  const [processing, setProcessing] = useState(false)
  const [qrCodeUrl, setQrCodeUrl] = useState<string | null>(null)
  const [orderId, setOrderId] = useState<string | null>(null)
  const [expireAt, setExpireAt] = useState<number>(0)

  const processWechatPayment = useCallback(
    async (topupAmount: number, payMethodIndex?: number) => {
      setProcessing(true)
      setQrCodeUrl(null)
      setOrderId(null)
      try {
        const response = await requestWechatPayment({
          amount: Math.floor(topupAmount),
          pay_method_index: payMethodIndex,
        })
        if (isApiSuccess(response)) {
          const result = getPayResult(response.data)
          if (result) {
            setQrCodeUrl(result.code_url)
            setOrderId(result.order_id)
            setExpireAt(result.expire_at)
            return result
          }
        }
        toast.error(getErrorMessage(response.message, response.data))
        return null
      } catch {
        toast.error(i18next.t('Payment request failed'))
        return null
      } finally {
        setProcessing(false)
      }
    },
    []
  )

  const resetWechatPayment = useCallback(() => {
    setQrCodeUrl(null)
    setOrderId(null)
    setExpireAt(0)
  }, [])

  return {
    processing,
    qrCodeUrl,
    orderId,
    expireAt,
    processWechatPayment,
    resetWechatPayment,
  }
}
