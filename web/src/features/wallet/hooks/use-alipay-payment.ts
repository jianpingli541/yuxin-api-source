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

import { requestAlipayPayment, isApiSuccess } from '../api'

interface AlipayPayResult {
  qr_code: string
  order_id: string
  expire_at: number
}

function getPayResult(data: unknown): AlipayPayResult | null {
  if (!data || typeof data !== 'object') return null
  const d = data as Record<string, unknown>
  if (typeof d.qr_code === 'string' && typeof d.order_id === 'string') {
    return {
      qr_code: d.qr_code,
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
 * Hook for handling Alipay (当面付 / scan-to-pay) payment processing.
 * Returns the qr_code string to render; caller polls order status.
 */
export function useAlipayPayment() {
  const [processing, setProcessing] = useState(false)
  const [qrCodeUrl, setQrCodeUrl] = useState<string | null>(null)
  const [orderId, setOrderId] = useState<string | null>(null)
  const [expireAt, setExpireAt] = useState<number>(0)

  const processAlipayPayment = useCallback(
    async (topupAmount: number, payMethodIndex?: number) => {
      setProcessing(true)
      setQrCodeUrl(null)
      setOrderId(null)
      try {
        const response = await requestAlipayPayment({
          amount: Math.floor(topupAmount),
          pay_method_index: payMethodIndex,
        })
        if (isApiSuccess(response)) {
          const result = getPayResult(response.data)
          if (result) {
            setQrCodeUrl(result.qr_code)
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

  const resetAlipayPayment = useCallback(() => {
    setQrCodeUrl(null)
    setOrderId(null)
    setExpireAt(0)
  }, [])

  return {
    processing,
    qrCodeUrl,
    orderId,
    expireAt,
    processAlipayPayment,
    resetAlipayPayment,
  }
}
