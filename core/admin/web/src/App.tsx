
import { BrowserRouter, Navigate, Route, Routes } from 'react-router-dom'
import { RequireAuth } from './components/RequireAuth'
import { AuthProvider } from './context/AuthContext'
import { OverviewStreamProvider } from './context/OverviewStreamContext'
import { AppLayout } from './layout/AppLayout'
import { CachePage } from './pages/CachePage'
import { ConfigPage } from './pages/ConfigPage'
import { LoginPage } from './pages/LoginPage'
import { LogsPage } from './pages/LogsPage'
import { OverviewPage } from './pages/OverviewPage'
import { RouteDetailPage } from './pages/RouteDetailPage'
import { RoutesPage } from './pages/RoutesPage'
import { ServicesPage } from './pages/ServicesPage'
import { ServiceDetailPage } from './pages/ServiceDetailPage'
import { EventsPage } from './pages/EventsPage'
import { InvestigatePage } from './pages/InvestigatePage'
import { SettingsPage } from './pages/SettingsPage'
import { TLSPage } from './pages/TLSPage'
import { TopologyPage } from './pages/TopologyPage'
import { HealthPage } from './pages/HealthPage'
import { WAFPage } from './pages/WAFPage'
import { MaintenancePage } from './pages/MaintenancePage'
import { ScenariosPage } from './pages/ScenariosPage'
import { TerminalPage } from './pages/TerminalPage'
import { JobsPage } from './pages/JobsPage'
import { RbacUsersPage } from './pages/rbac/RbacUsersPage'
import { RbacRolesPage } from './pages/rbac/RbacRolesPage'
import { RbacPermissionsPage } from './pages/rbac/RbacPermissionsPage'
import { MessagesPage } from './pages/MessagesPage'

export default function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route
            path="/"
            element={
              <RequireAuth />
            }
          >
            <Route
              element={
                <OverviewStreamProvider>
                  <AppLayout />
                </OverviewStreamProvider>
              }
            >
              <Route index element={<OverviewPage />} />
              <Route path="attention" element={<Navigate to="/events" replace />} />
              <Route path="routes" element={<RoutesPage />} />
              <Route path="routes/:ruleIndex/:pathIndex" element={<RouteDetailPage />} />
              <Route path="services" element={<ServicesPage />} />
              <Route path="services/:name" element={<ServiceDetailPage />} />
              <Route path="topology" element={<TopologyPage />} />
              <Route path="healths" element={<HealthPage />} />
              <Route path="health" element={<Navigate to="/healths" replace />} />
              <Route path="cache" element={<CachePage />} />
              <Route path="waf" element={<WAFPage />} />
              <Route path="maintenance" element={<MaintenancePage />} />
              <Route path="scenarios" element={<ScenariosPage />} />
              <Route path="terminal" element={<TerminalPage />} />
              <Route path="tls" element={<TLSPage />} />
              <Route path="config" element={<ConfigPage />} />
              <Route path="jobs" element={<JobsPage />} />
              <Route path="rbac/users" element={<RbacUsersPage />} />
              <Route path="rbac/roles" element={<RbacRolesPage />} />
              <Route path="rbac/permissions" element={<RbacPermissionsPage />} />
              <Route path="settings" element={<SettingsPage />} />
              <Route path="logs" element={<LogsPage />} />
              <Route path="events" element={<EventsPage />} />
              <Route path="investigate" element={<InvestigatePage />} />
              <Route path="messages" element={<MessagesPage />} />
              <Route path="*" element={<Navigate to="/" replace />} />
            </Route>
          </Route>
        </Routes>
      </AuthProvider>
    </BrowserRouter>
  )
}
