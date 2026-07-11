import { Navigate, Route, Routes, useParams } from 'react-router-dom'
import { AppLayout } from '@/layouts/AppLayout'
import { ErrorBoundary } from '@/components/shared/ErrorBoundary'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { HostDetailPage, HostsPage } from '@/features/hosts/HostsPage'
import { BackupDetailPage, BackupsPage } from '@/features/backups/BackupsPage'
import { OperationDetailPage, OperationsPage } from '@/features/operations/OperationsPage'
import { TaskDetailPage, TasksPage } from '@/features/tasks/TasksPage'
import { NotificationDetailPage, NotificationsPage } from '@/features/notifications/NotificationsPage'
import { TemplatesPage } from '@/features/templates/TemplatesPage'
import { StorageLayout } from '@/features/storage/StorageLayout'
import { LocalStoragePage } from '@/features/storage/LocalStoragePage'
import { RemoteStoragePage } from '@/features/storage/RemoteStoragePage'
import { AboutPage } from '@/features/settings/SettingsPage'
import { SettingsLayout } from '@/features/settings/SettingsLayout'
import { GeneralTab } from '@/features/settings/tabs/GeneralTab'
import { DestinationsTab } from '@/features/settings/tabs/DestinationsTab'
import { SyncTab } from '@/features/settings/tabs/SyncTab'
import { VaultTab } from '@/features/settings/tabs/VaultTab'
import { AuditTab } from '@/features/settings/tabs/AuditTab'
import { LogsTab } from '@/features/settings/tabs/LogsTab'

function HostRoute() {
  const { id = '' } = useParams()
  return <HostDetailPage id={id} />
}

function BackupRoute() {
  const { id = '' } = useParams()
  return <BackupDetailPage id={id} />
}

function OperationRoute() {
  const { id = '' } = useParams()
  return <OperationDetailPage id={id} />
}

function TaskRoute() {
  const { id = '' } = useParams()
  return <TaskDetailPage id={id} />
}

function NotificationRoute() {
  const { id = '' } = useParams()
  return <NotificationDetailPage id={id} />
}

function withBoundary(element: React.ReactNode, title?: string) {
  return <ErrorBoundary fallbackTitle={title}>{element}</ErrorBoundary>
}

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={withBoundary(<DashboardPage />)} />
        <Route path="hosts" element={withBoundary(<HostsPage />)} />
        <Route path="hosts/:id" element={withBoundary(<HostRoute />, 'Host error')} />
        <Route path="backups" element={withBoundary(<BackupsPage />)} />
        <Route path="backups/:id" element={withBoundary(<BackupRoute />, 'Backup error')} />
        <Route path="storage" element={withBoundary(<StorageLayout />)}>
          <Route index element={<Navigate to="local" replace />} />
          <Route path="local" element={<LocalStoragePage />} />
          <Route path="remote" element={<RemoteStoragePage />} />
        </Route>
        <Route path="operations" element={withBoundary(<OperationsPage />)} />
        <Route path="operations/:id" element={withBoundary(<OperationRoute />, 'Operation error')} />
        <Route path="tasks" element={withBoundary(<TasksPage />)} />
        <Route path="tasks/:id" element={withBoundary(<TaskRoute />, 'Task error')} />
        <Route path="notifications" element={withBoundary(<NotificationsPage />)} />
        <Route path="notifications/:id" element={withBoundary(<NotificationRoute />, 'Notification error')} />
        <Route path="templates" element={withBoundary(<TemplatesPage />)} />
        <Route path="settings" element={withBoundary(<SettingsLayout />)}>
          <Route index element={<Navigate to="general" replace />} />
          <Route path="general" element={<GeneralTab />} />
          <Route path="destinations" element={<DestinationsTab />} />
          <Route path="sync" element={<SyncTab />} />
          <Route path="vault" element={<VaultTab />} />
          <Route path="audit" element={<AuditTab />} />
          <Route path="logs" element={<LogsTab />} />
        </Route>
        <Route path="about" element={withBoundary(<AboutPage />)} />
        <Route path="*" element={<p className="text-sm">Page not found</p>} />
      </Route>
    </Routes>
  )
}
