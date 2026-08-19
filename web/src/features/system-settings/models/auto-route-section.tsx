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
import { zodResolver } from '@hookform/resolvers/zod'
import { useMemo, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useResetForm } from '../hooks/use-reset-form'
import { useUpdateOption } from '../hooks/use-update-option'

const autoRouteSchema = z.object({
  'auto_route.enabled': z.boolean(),
  'auto_route.aliases': z.string().refine((value) => {
    const trimmed = value.trim()
    if (!trimmed) return true
    try {
      const parsed = JSON.parse(trimmed)
      if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
        return false
      }
      return Object.values(parsed).every(
        (pool) =>
          Array.isArray(pool) &&
          pool.every((m) => typeof m === 'string' && m.trim().length > 0)
      )
    } catch {
      return false
    }
  }, 'Aliases must be a JSON object: { "alias": ["model-a", "model-b"] }'),
  'auto_route.doubao_boost': z.coerce
    .number()
    .min(0, 'Boost must be >= 0')
    .max(1, 'Boost must be <= 1 (100%)'),
})

type AutoRouteFormValues = z.infer<typeof autoRouteSchema>

type AutoRouteSectionProps = {
  defaultValues: AutoRouteFormValues
}

export function AutoRouteSection({ defaultValues }: AutoRouteSectionProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const baselineRef = useRef<AutoRouteFormValues>(defaultValues)

  const formDefaults = useMemo(() => defaultValues, [defaultValues])

  const form = useForm<AutoRouteFormValues>({
    resolver: zodResolver(autoRouteSchema),
    defaultValues: formDefaults,
  })

  useResetForm(form, formDefaults)

  const onSubmit = async (values: AutoRouteFormValues) => {
    const updates = (
      Object.keys(values) as Array<keyof AutoRouteFormValues>
    ).filter((key) => values[key] !== baselineRef.current[key])

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of updates) {
      await updateOption.mutateAsync({ key, value: values[key] })
    }

    baselineRef.current = values
  }

  return (
    <SettingsSection title={t('Smart Routing (auto model)')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />

          <div className='flex min-w-0 flex-col gap-6'>
            <FormField
              control={form.control}
              name='auto_route.enabled'
              render={({ field }) => (
                <SettingsSwitchItem>
                  <SettingsSwitchContent>
                    <FormLabel>{t('Enable smart routing')}</FormLabel>
                    <FormDescription>
                      {t(
                        'Allow clients to send model aliases (e.g. "auto"); the gateway picks the best (model, channel) combo by quality + speed.'
                      )}
                    </FormDescription>
                  </SettingsSwitchContent>
                  <FormControl>
                    <Switch
                      checked={field.value}
                      onCheckedChange={field.onChange}
                    />
                  </FormControl>
                </SettingsSwitchItem>
              )}
            />

            <FormField
              control={form.control}
              name='auto_route.aliases'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Alias pools (JSON)')}</FormLabel>
                  <FormControl>
                    <Textarea
                      className='min-h-32 font-mono text-sm'
                      placeholder='{"auto": ["doubao-seed-1.6", "kimi-k2"]}'
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Map a client-facing alias to candidate models. Candidates must be enabled on channels available to the caller group. Billed by the actually served model; the X-Yuxin-Model response header reports it.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name='auto_route.doubao_boost'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('Doubao ecosystem boost')}</FormLabel>
                  <FormControl>
                    <Input type='number' step='0.01' min='0' max='1' {...field} />
                  </FormControl>
                  <FormDescription>
                    {t(
                      'Score multiplier for VolcEngine/Doubao channels (0.15 = +15%). Lets new ByteDance-ecosystem models win ties so customers taste them first. 0 disables the bias.'
                    )}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
