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

export interface AlipaySettingsValues {
  AlipayEnabled: boolean
  AlipayAppId: string
  AlipayPrivateKey: string
  AlipayPublicKey: string
  AlipayUseCertMode: boolean
  AlipayAppCertPublicKey: string
  AlipayPublicCert: string
  AlipayRootCert: string
  AlipaySandbox: boolean
  AlipayNotifyUrl: string
  AlipayReturnUrl: string
  AlipayUnitPrice: number
  AlipayMinTopUp: number
}

type AlipayFieldValues = AlipaySettingsValues

interface Props {
  values: AlipaySettingsValues
  onValueChange: <K extends keyof AlipayFieldValues>(
    key: K,
    value: AlipayFieldValues[K]
  ) => void
}

export function AlipaySettingsSection({ values, onValueChange }: Props) {
  const { t } = useTranslation()

  return (
    <div className='space-y-4 pt-4'>
      <div>
        <h3 className='text-lg font-medium'>{t('Alipay (Face-to-Face Payment)')}</h3>
        <p className='text-muted-foreground text-sm'>
          {t(
            'Alipay in-person payment via precreate QR code. Users scan the code with the Alipay app to complete the top-up.'
          )}
        </p>
      </div>
      <Alert>
        <AlertDescription className='text-xs'>
          {values.AlipayUseCertMode
            ? t(
                'Certificate mode: upload the application public key cert, Alipay public key cert, and Alipay root cert (all PEM). Recommended for production.'
              )
            : t(
                'Public key mode: enter the Alipay public key (for async notification signature verification). Simpler setup.'
              )}
        </AlertDescription>
      </Alert>

      <div className='grid gap-4 sm:grid-cols-2'>
        <SettingsSwitchField
          checked={values.AlipayEnabled}
          onCheckedChange={(v) => onValueChange('AlipayEnabled', v)}
          label={t('Enable Alipay')}
          className='py-0'
        />
        <SettingsSwitchField
          checked={values.AlipaySandbox}
          onCheckedChange={(v) => onValueChange('AlipaySandbox', v)}
          label={t('Sandbox mode')}
          className='py-0'
        />
      </div>

      <div className='grid gap-1.5'>
        <Label>{t('AppID')}</Label>
        <Input
          value={values.AlipayAppId}
          onChange={(event) =>
            onValueChange('AlipayAppId', event.target.value)
          }
        />
      </div>

      <SettingsSwitchField
        checked={values.AlipayUseCertMode}
        onCheckedChange={(v) => onValueChange('AlipayUseCertMode', v)}
        label={t('Certificate mode (production-recommended; uses 3 PEM certs instead of public key)')}
      />

      {values.AlipayUseCertMode ? (
        <div className='space-y-4'>
          <div className='grid gap-1.5'>
            <Label>{t('Application Public Key Cert (PEM)')}</Label>
            <Textarea
              rows={6}
              value={values.AlipayAppCertPublicKey}
              onChange={(event) =>
                onValueChange('AlipayAppCertPublicKey', event.target.value)
              }
              className='font-mono text-xs'
              placeholder='-----BEGIN CERTIFICATE-----'
            />
            <p className='text-muted-foreground text-xs'>
              {t('应用公钥证书 — 从开放平台上传应用公钥后下载。')}
            </p>
          </div>
          <div className='grid gap-1.5'>
            <Label>{t('Alipay Public Key Cert (PEM)')}</Label>
            <Textarea
              rows={6}
              value={values.AlipayPublicCert}
              onChange={(event) =>
                onValueChange('AlipayPublicCert', event.target.value)
              }
              className='font-mono text-xs'
              placeholder='-----BEGIN CERTIFICATE-----'
            />
            <p className='text-muted-foreground text-xs'>
              {t('支付宝公钥证书 — 用于验签异步通知。')}
            </p>
          </div>
          <div className='grid gap-1.5'>
            <Label>{t('Alipay Root Cert (PEM)')}</Label>
            <Textarea
              rows={6}
              value={values.AlipayRootCert}
              onChange={(event) =>
                onValueChange('AlipayRootCert', event.target.value)
              }
              className='font-mono text-xs'
              placeholder='-----BEGIN CERTIFICATE-----'
            />
            <p className='text-muted-foreground text-xs'>
              {t('支付宝根证书 — 完整信任链。')}
            </p>
          </div>
        </div>
      ) : (
        <div className='grid grid-cols-2 gap-4'>
          <div className='grid gap-1.5'>
            <Label>{t('Application Private Key (PEM)')}</Label>
            <Textarea
              rows={6}
              value={values.AlipayPrivateKey}
              onChange={(event) =>
                onValueChange('AlipayPrivateKey', event.target.value)
              }
              className='font-mono text-xs'
              placeholder='-----BEGIN RSA PRIVATE KEY-----'
            />
            <p className='text-muted-foreground text-xs'>
              {t(
                'RSA2 private key from the Alipay key generator — used to sign API requests.'
              )}
            </p>
          </div>
          <div className='grid gap-1.5'>
            <Label>{t('Alipay Public Key (PEM)')}</Label>
            <Textarea
              rows={6}
              value={values.AlipayPublicKey}
              onChange={(event) =>
                onValueChange('AlipayPublicKey', event.target.value)
              }
              className='font-mono text-xs'
              placeholder='-----BEGIN PUBLIC KEY-----'
            />
            <p className='text-muted-foreground text-xs'>
              {t(
                'Alipay public key (not application public key) — used to verify async notification signatures.'
              )}
            </p>
          </div>
        </div>
      )}

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
            value={values.AlipayUnitPrice}
            onChange={(event) =>
              onValueChange(
                'AlipayUnitPrice',
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
            value={values.AlipayMinTopUp}
            onChange={(event) =>
              onValueChange(
                'AlipayMinTopUp',
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
            placeholder='https://ai.yuxin.yun/api/alipay/notify'
            value={values.AlipayNotifyUrl}
            onChange={(event) =>
              onValueChange('AlipayNotifyUrl', event.target.value)
            }
          />
        </div>
        <div className='grid gap-1.5'>
          <Label>{t('Payment return URL')}</Label>
          <Input
            placeholder='https://ai.yuxin.yun/wallet'
            value={values.AlipayReturnUrl}
            onChange={(event) =>
              onValueChange('AlipayReturnUrl', event.target.value)
            }
          />
        </div>
      </div>
    </div>
  )
}
