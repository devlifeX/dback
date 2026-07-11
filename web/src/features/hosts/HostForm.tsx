import { useForm, useFieldArray, Controller } from 'react-hook-form'
import { Button } from '@/components/ui/button'
import { Input, Textarea } from '@/components/ui/input'
import { Checkbox } from '@/components/ui/checkbox'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { FormField } from '@/components/forms/FormField'
import { FormSection } from '@/components/forms/FormSection'
import { SecretField } from '@/components/forms/SecretField'
import type { Profile } from '@/api/types'
import { emptyProfile } from '@/api/types'
import { useQuery } from '@tanstack/react-query'
import { destinationsApi } from '@/api/settings'

export function HostForm({
  profile,
  onSubmit,
  onCancel,
  pending,
}: {
  profile?: Profile
  onSubmit: (values: Profile) => void
  onCancel: () => void
  pending?: boolean
}) {
  const destinations = useQuery({ queryKey: ['destinations'], queryFn: destinationsApi.list })
  const destItems = destinations.data?.items ?? []

  const { register, handleSubmit, control, watch, setValue } = useForm<Profile>({
    defaultValues: profile ?? emptyProfile(),
  })

  const connectionType = watch('connection_type')
  const fileBackupEnabled = watch('file_backup_enabled')
  const authType = watch('auth_type')
  const jumpAuthType = watch('jump_auth_type')

  const paths = useFieldArray({ control, name: 'file_backup_paths' })
  const excludeValues = watch('file_backup_exclude') ?? []

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2">
        <FormField label="Name" htmlFor="host-name">
          <Input id="host-name" {...register('name', { required: true })} />
        </FormField>
        <FormField label="Group" htmlFor="host-group">
          <Input id="host-group" {...register('group')} placeholder="Default" />
        </FormField>
      </div>

      <Tabs defaultValue="connection">
        <TabsList className="flex flex-wrap h-auto gap-1">
          <TabsTrigger value="connection">Connection</TabsTrigger>
          <TabsTrigger value="database">Database</TabsTrigger>
          <TabsTrigger value="filebackup">File backup</TabsTrigger>
          <TabsTrigger value="upload">Remote upload</TabsTrigger>
          <TabsTrigger value="queries">Queries</TabsTrigger>
          <TabsTrigger value="security">Security</TabsTrigger>
        </TabsList>

        <TabsContent value="connection">
          <FormSection title="Connection" description="SSH, jump host, localhost, or WordPress">
            <FormField label="Type" className="sm:col-span-2">
              <Controller
                control={control}
                name="connection_type"
                render={({ field }) => (
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="SSH">SSH</SelectItem>
                      <SelectItem value="JumpHost">Jump host</SelectItem>
                      <SelectItem value="Localhost">Localhost</SelectItem>
                      <SelectItem value="WordPress">WordPress</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              />
            </FormField>

            {connectionType === 'WordPress' ? (
              <>
                <FormField label="WordPress URL" htmlFor="wp-url">
                  <Input id="wp-url" {...register('wp_url')} placeholder="https://example.com" />
                </FormField>
                <SecretField label="API key" value={watch('wp_key') ?? ''} onChange={(v) => setValue('wp_key', v)} hint="Generate from host detail page" />
              </>
            ) : (
              <>
                <FormField label="Host" htmlFor="host-host">
                  <Input id="host-host" {...register('host')} />
                </FormField>
                <FormField label="Port" htmlFor="host-port">
                  <Input id="host-port" {...register('port')} />
                </FormField>
                {connectionType !== 'Localhost' ? (
                  <>
                    <FormField label="SSH user" htmlFor="ssh-user">
                      <Input id="ssh-user" {...register('ssh_user')} />
                    </FormField>
                    <FormField label="Auth type">
                      <Controller
                        control={control}
                        name="auth_type"
                        render={({ field }) => (
                          <Select value={field.value} onValueChange={field.onChange}>
                            <SelectTrigger><SelectValue /></SelectTrigger>
                            <SelectContent>
                              <SelectItem value="Password">Password</SelectItem>
                              <SelectItem value="Key File">Key file</SelectItem>
                            </SelectContent>
                          </Select>
                        )}
                      />
                    </FormField>
                    {authType === 'Password' ? (
                      <SecretField label="SSH password" value={watch('ssh_password') ?? ''} onChange={(v) => setValue('ssh_password', v)} />
                    ) : (
                      <>
                        <FormField label="Key path" htmlFor="auth-key-path">
                          <Input id="auth-key-path" {...register('auth_key_path')} />
                        </FormField>
                        <SecretField label="Key PEM" value={watch('auth_key_pem') ?? ''} onChange={(v) => setValue('auth_key_pem', v)} />
                      </>
                    )}
                  </>
                ) : null}
              </>
            )}

            {connectionType === 'JumpHost' ? (
              <>
                <FormField label="Jump host" htmlFor="jump-host">
                  <Input id="jump-host" {...register('jump_host')} />
                </FormField>
                <FormField label="Jump port" htmlFor="jump-port">
                  <Input id="jump-port" {...register('jump_port')} />
                </FormField>
                <FormField label="Jump user" htmlFor="jump-user">
                  <Input id="jump-user" {...register('jump_user')} />
                </FormField>
                <FormField label="Jump auth type">
                  <Controller
                    control={control}
                    name="jump_auth_type"
                    render={({ field }) => (
                      <Select value={field.value ?? 'Password'} onValueChange={field.onChange}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="Password">Password</SelectItem>
                          <SelectItem value="Key File">Key file</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  />
                </FormField>
                {jumpAuthType === 'Key File' ? (
                  <SecretField label="Jump key PEM" value={watch('jump_auth_key_pem') ?? ''} onChange={(v) => setValue('jump_auth_key_pem', v)} />
                ) : (
                  <SecretField label="Jump password" value={watch('jump_password') ?? ''} onChange={(v) => setValue('jump_password', v)} />
                )}
              </>
            ) : null}
          </FormSection>
        </TabsContent>

        <TabsContent value="database">
          <FormSection title="Database" description="MySQL/MariaDB connection settings">
            <FormField label="DB host" htmlFor="db-host">
              <Input id="db-host" {...register('db_host')} />
            </FormField>
            <FormField label="DB port" htmlFor="db-port">
              <Input id="db-port" {...register('db_port')} />
            </FormField>
            <FormField label="DB user" htmlFor="db-user">
              <Input id="db-user" {...register('db_user')} />
            </FormField>
            <SecretField label="DB password" value={watch('db_password') ?? ''} onChange={(v) => setValue('db_password', v)} />
            <FormField label="DB type">
              <Controller
                control={control}
                name="db_type"
                render={({ field }) => (
                  <Select value={field.value} onValueChange={field.onChange}>
                    <SelectTrigger><SelectValue /></SelectTrigger>
                    <SelectContent>
                      <SelectItem value="MySQL">MySQL</SelectItem>
                      <SelectItem value="MariaDB">MariaDB</SelectItem>
                    </SelectContent>
                  </Select>
                )}
              />
            </FormField>
            <FormField label="Target database" htmlFor="target-db">
              <Input id="target-db" {...register('target_db_name')} />
            </FormField>
            <FormField label="Destination path" htmlFor="destination">
              <Input id="destination" {...register('destination')} />
            </FormField>
            <div className="flex items-center gap-2 sm:col-span-2">
              <Controller
                control={control}
                name="is_docker"
                render={({ field }) => (
                  <Checkbox checked={field.value} onCheckedChange={field.onChange} />
                )}
              />
              <span className="text-sm">Docker container</span>
            </div>
            {watch('is_docker') ? (
              <FormField label="Container ID" htmlFor="container-id">
                <Input id="container-id" {...register('container_id')} />
              </FormField>
            ) : null}
          </FormSection>
        </TabsContent>

        <TabsContent value="filebackup">
          <FormSection title="File backup" description="Archive remote paths over SSH">
            <div className="flex items-center gap-2 sm:col-span-2">
              <Controller
                control={control}
                name="file_backup_enabled"
                render={({ field }) => (
                  <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
                )}
              />
              <span className="text-sm">Enable file backup</span>
            </div>
            {fileBackupEnabled ? (
              <>
                <FormField label="Destination" htmlFor="fb-dest">
                  <Input id="fb-dest" {...register('file_backup_destination')} />
                </FormField>
                <FormField label="Compression">
                  <Controller
                    control={control}
                    name="file_backup_compression"
                    render={({ field }) => (
                      <Select value={field.value ?? 'zstd'} onValueChange={field.onChange}>
                        <SelectTrigger><SelectValue /></SelectTrigger>
                        <SelectContent>
                          <SelectItem value="zstd">Zstd</SelectItem>
                          <SelectItem value="gzip">Gzip</SelectItem>
                        </SelectContent>
                      </Select>
                    )}
                  />
                </FormField>
                <div className="sm:col-span-2 space-y-2">
                  <p className="text-sm font-medium">Paths</p>
                  {paths.fields.map((f, i) => (
                    <div key={f.id} className="flex gap-2">
                      <Input placeholder="Name" {...register(`file_backup_paths.${i}.name`)} />
                      <Input placeholder="Remote path" {...register(`file_backup_paths.${i}.remote_path`)} />
                      <Button type="button" variant="outline" size="sm" onClick={() => paths.remove(i)}>Remove</Button>
                    </div>
                  ))}
                  <Button type="button" variant="outline" size="sm" onClick={() => paths.append({ id: crypto.randomUUID(), name: '', remote_path: '' })}>
                    Add path
                  </Button>
                </div>
                <div className="sm:col-span-2 space-y-2">
                  <p className="text-sm font-medium">Exclude patterns</p>
                  {excludeValues.map((_, i) => (
                    <div key={i} className="flex gap-2">
                      <Input {...register(`file_backup_exclude.${i}`)} />
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={() => setValue('file_backup_exclude', excludeValues.filter((_, j) => j !== i))}
                      >
                        Remove
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => setValue('file_backup_exclude', [...excludeValues, ''])}
                  >
                    Add exclude
                  </Button>
                </div>
              </>
            ) : null}
          </FormSection>
        </TabsContent>

        <TabsContent value="upload">
          <FormSection title="Remote upload" description="Auto-upload backups to remote destinations">
            <div className="flex items-center gap-2">
              <Controller control={control} name="remote_auto_upload_db" render={({ field }) => (
                <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
              )} />
              <span className="text-sm">Auto-upload database backups</span>
            </div>
            <div className="flex items-center gap-2">
              <Controller control={control} name="remote_auto_upload_files" render={({ field }) => (
                <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
              )} />
              <span className="text-sm">Auto-upload file backups</span>
            </div>
            <div className="sm:col-span-2 space-y-2">
              <p className="text-sm font-medium">Destinations</p>
              {destItems.map((d) => {
                const ids = watch('remote_upload_destination_ids') ?? []
                const checked = ids.includes(d.id)
                return (
                  <label key={d.id} className="flex items-center gap-2 text-sm">
                    <Checkbox
                      checked={checked}
                      onCheckedChange={(c) => {
                        const next = c ? [...ids, d.id] : ids.filter((x) => x !== d.id)
                        setValue('remote_upload_destination_ids', next)
                      }}
                    />
                    {d.name}
                  </label>
                )
              })}
              {destItems.length === 0 ? (
                <p className="text-xs text-[hsl(var(--muted-foreground))]">No destinations configured in Settings.</p>
              ) : null}
            </div>
          </FormSection>
        </TabsContent>

        <TabsContent value="queries">
          <FormSection title="Import queries" description="SQL run before/after restore">
            <FormField label="Pre-import query" className="sm:col-span-2">
              <Textarea rows={4} className="font-mono text-xs" {...register('pre_import_query')} />
            </FormField>
            <div className="flex items-center gap-2">
              <Controller control={control} name="run_query_before_import" render={({ field }) => (
                <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
              )} />
              <span className="text-sm">Run before import</span>
            </div>
            <FormField label="Post-import query" className="sm:col-span-2">
              <Textarea rows={4} className="font-mono text-xs" {...register('post_import_query')} />
            </FormField>
            <div className="flex items-center gap-2">
              <Controller control={control} name="run_query_after_import" render={({ field }) => (
                <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
              )} />
              <span className="text-sm">Run after import</span>
            </div>
          </FormSection>
        </TabsContent>

        <TabsContent value="security">
          <FormSection title="Security" description="Protect production hosts from restore">
            <div className="flex items-center gap-2 sm:col-span-2">
              <Controller control={control} name="import_protected" render={({ field }) => (
                <Checkbox checked={!!field.value} onCheckedChange={field.onChange} />
              )} />
              <span className="text-sm">Import protected (block restore to this host)</span>
            </div>
          </FormSection>
        </TabsContent>
      </Tabs>

      <div className="flex justify-end gap-2 border-t border-[hsl(var(--border))] pt-4">
        <Button type="button" variant="outline" onClick={onCancel}>Cancel</Button>
        <Button type="submit" disabled={pending}>{profile?.id ? 'Save host' : 'Create host'}</Button>
      </div>
    </form>
  )
}
