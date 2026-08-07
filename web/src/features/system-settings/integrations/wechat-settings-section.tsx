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
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { SettingsSwitchField } from '../components/settings-form-layout'

export interface WechatSettingsValues {
  WechatEnabled: boolean
  WechatMerchantId: string
  WechatAppId: string
  WechatApiV3Key: string
  WechatPrivateKey: string
  WechatCertSerialNo: string
  // 商户 API 证书 apiclient_cert.pem 的 PEM 内容（可选）。
  // 上传后服务端自动从证书中提取序列号；为空时回退手工序列号。
  WechatCertPublicKey: string
  WechatNotifyUrl: string
  WechatReturnUrl: string
  WechatUnitPrice: number
  WechatMinTopUp: number
}

type WechatFieldValues = WechatSettingsValues

interface Props {
  values: WechatSettingsValues
  onValueChange: <K extends keyof WechatFieldValues>(
    key: K,
    value: WechatFieldValues[K]
  ) => void
}

export function WechatSettingsSection({ values, onValueChange }: Props) {
  const { t } = useTranslation()

  return (
    <div className='space-y-4 pt-4'>
      <div>
        <h3 className='text-lg font-medium'>{t('WeChat Pay (Native)')}</h3>
        <p className='text-muted-foreground text-sm'>
          {t(
            'WeChat Native payment (scan-to-pay on PC). Users scan a QR code with the WeChat app to complete the top-up.'
          )}
        </p>
      </div>
      <Alert>
        <AlertDescription className='text-xs'>
          {t(
            'Obtain the merchant ID, AppID, API V3 key, merchant private key, and certificate serial number from the WeChat Pay merchant platform.'
          )}
        </AlertDescription>
      </Alert>

      <SettingsSwitchField
        checked={values.WechatEnabled}
        onCheckedChange={(v) => onValueChange('WechatEnabled', v)}
        label={t('Enable WeChat Pay')}
        className='py-0'
      />

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Merchant ID (mchid)')}</Label>
          <Input
            value={values.WechatMerchantId}
            onChange={(event) =>
              onValueChange('WechatMerchantId', event.target.value)
            }
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('AppID')}</Label>
          <Input
            value={values.WechatAppId}
            onChange={(event) =>
              onValueChange('WechatAppId', event.target.value)
            }
          />
        </div>
      </div>

      <div className='grid gap-1.5'>
        <Label>{t('API V3 Key')}</Label>
        <Input
          type='password'
          value={values.WechatApiV3Key}
          onChange={(event) =>
            onValueChange('WechatApiV3Key', event.target.value)
          }
        />
        <p className='text-muted-foreground text-xs'>
          {t(
            'API V3 key (32-byte), used to decrypt callback notifications with AES-256-GCM.'
          )}
        </p>
      </div>

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Merchant Private Key (PEM)')}</Label>
          <Textarea
            rows={6}
            value={values.WechatPrivateKey}
            onChange={(event) =>
              onValueChange('WechatPrivateKey', event.target.value)
            }
            className='font-mono text-xs'
            placeholder='-----BEGIN PRIVATE KEY-----'
          />
          <p className='text-muted-foreground text-xs'>
            {t('apiclient_key.pem — used to sign API requests.')}
          </p>
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Certificate Serial Number')}</Label>
          <Input
            value={values.WechatCertSerialNo}
            onChange={(event) =>
              onValueChange('WechatCertSerialNo', event.target.value)
            }
          />
          <p className='text-muted-foreground text-xs'>
            {t('Certificate serial number from the WeChat Pay merchant platform.')}
          </p>
        </div>
      </div>

      <div className='grid gap-1.5'>
        <Label>{t('微信支付证书（apiclient_cert.pem）')}</Label>
        <Textarea
          rows={6}
          value={values.WechatCertPublicKey}
          onChange={(event) =>
            onValueChange('WechatCertPublicKey', event.target.value)
          }
          className='font-mono text-xs'
          placeholder='-----BEGIN CERTIFICATE-----'
        />
        <p className='text-muted-foreground text-xs'>
          {t(
            'Merchant certificate (apiclient_cert.pem) — serial number is auto-extracted; alternatively fill the certificate serial number field manually.'
          )}
        </p>
      </div>

      <div className='grid grid-cols-3 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Currency')}</Label>
          <Input value='CNY' disabled />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Unit price (CNY)')}</Label>
          <Input
            type='number'
            step={0.1}
            min={0}
            value={values.WechatUnitPrice}
            onChange={(event) =>
              onValueChange(
                'WechatUnitPrice',
                event.target.value === '' ? 0 : event.target.valueAsNumber
              )
            }
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Minimum top-up quantity')}</Label>
          <Input
            type='number'
            min={1}
            value={values.WechatMinTopUp}
            onChange={(event) =>
              onValueChange(
                'WechatMinTopUp',
                event.target.value === '' ? 1 : event.target.valueAsNumber
              )
            }
          />
        </div>
      </div>

      <div className='grid grid-cols-2 gap-4'>
        <div className='grid gap-1.5'>
          <Label>{t('Callback notification URL')}</Label>
          <Input
            placeholder='https://ai.yuxin.yun/api/wechat/notify'
            value={values.WechatNotifyUrl}
            onChange={(event) =>
              onValueChange('WechatNotifyUrl', event.target.value)
            }
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Payment return URL')}</Label>
          <Input
            placeholder='https://ai.yuxin.yun/wallet'
            value={values.WechatReturnUrl}
            onChange={(event) =>
              onValueChange('WechatReturnUrl', event.target.value)
            }
          />
        </div>
      </div>
    </div>
  )
}
