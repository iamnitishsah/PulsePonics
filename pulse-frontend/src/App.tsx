import { lazy, Suspense } from 'react'
import { BrowserRouter, Route, Routes } from 'react-router'
import { AuthProvider } from './auth/AuthProvider'
import { Layout } from './components/Layout'
import { PublicOnly, RequireAuth } from './components/RouteGuards'
import { AccountPage } from './pages/AccountPage'
import { DeviceSettingsPage } from './pages/DeviceSettingsPage'
import { DevicesPage } from './pages/DevicesPage'
import { LoginPage, SignupPage } from './pages/AuthPages'
import { NotFoundPage } from './pages/NotFoundPage'
import { PageSpinner } from './components/ui/Spinner'

// The device page carries the charting library; load it only when needed.
const DevicePage = lazy(() => import('./pages/DevicePage').then((m) => ({ default: m.DevicePage })))

export default function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Routes>
          <Route element={<PublicOnly />}>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/signup" element={<SignupPage />} />
          </Route>
          <Route element={<RequireAuth />}>
            <Route element={<Layout />}>
              <Route index element={<DevicesPage />} />
              <Route
                path="/devices/:deviceId"
                element={
                  <Suspense fallback={<PageSpinner label="Loading device" />}>
                    <DevicePage />
                  </Suspense>
                }
              />
              <Route path="/devices/:deviceId/settings" element={<DeviceSettingsPage />} />
              <Route path="/account" element={<AccountPage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Route>
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthProvider>
  )
}
