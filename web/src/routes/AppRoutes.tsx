import { Navigate, Route, Routes, useParams } from 'react-router-dom'
import { AppLayout } from '@/layouts/AppLayout'
import { DashboardPage } from '@/features/dashboard/DashboardPage'
import { HostDetailPage, HostsPage } from '@/features/hosts/HostsPage'
import { OperationDetailPage, OperationsPage } from '@/features/operations/OperationsPage'
import { TaskDetailPage, TasksPage } from '@/features/tasks/TasksPage'
import { NotificationsPage } from '@/features/notifications/NotificationsPage'
import { TemplatesPage } from '@/features/templates/TemplatesPage'
import { AboutPage, SettingsPage } from '@/features/settings/SettingsPage'

function HostRoute() {
  const { id = '' } = useParams()
  return <HostDetailPage id={id} />
}

function OperationRoute() {
  const { id = '' } = useParams()
  return <OperationDetailPage id={id} />
}

function TaskRoute() {
  const { id = '' } = useParams()
  return <TaskDetailPage id={id} />
}

export function AppRoutes() {
  return (
    <Routes>
      <Route element={<AppLayout />}>
        <Route index element={<Navigate to="/dashboard" replace />} />
        <Route path="dashboard" element={<DashboardPage />} />
        <Route path="hosts" element={<HostsPage />} />
        <Route path="hosts/:id" element={<HostRoute />} />
        <Route path="operations" element={<OperationsPage />} />
        <Route path="operations/:id" element={<OperationRoute />} />
        <Route path="tasks" element={<TasksPage />} />
        <Route path="tasks/:id" element={<TaskRoute />} />
        <Route path="notifications" element={<NotificationsPage />} />
        <Route path="templates" element={<TemplatesPage />} />
        <Route path="settings" element={<SettingsPage />} />
        <Route path="about" element={<AboutPage />} />
        <Route path="*" element={<p className="text-sm">Page not found</p>} />
      </Route>
    </Routes>
  )
}
