import { useState, type FormEvent } from 'react'
import { ApiError, api, errorMessage } from '../lib/api'
import type { DeviceWithKey } from '../types/api'
import { Button } from './ui/Button'
import { Modal } from './ui/Modal'
import { Notice } from './ui/Notice'
import { TextField } from './ui/TextField'

const DEVICE_ID_PATTERN = /^[A-Za-z0-9_-]{3,64}$/

export function AddDeviceDialog({
  open,
  onClose,
  onCreated,
}: {
  open: boolean
  onClose: () => void
  onCreated: (device: DeviceWithKey) => void
}) {
  const [deviceId, setDeviceId] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState<unknown>(null)
  const [busy, setBusy] = useState(false)

  const trimmed = deviceId.trim()
  const formatError = trimmed && !DEVICE_ID_PATTERN.test(trimmed) ? '3–64 characters: letters, digits, - or _' : null
  const apiFieldError = error instanceof ApiError && error.field === 'device_id' ? error.message : null

  const close = () => {
    setDeviceId('')
    setName('')
    setError(null)
    onClose()
  }

  const submit = async (e: FormEvent) => {
    e.preventDefault()
    if (!trimmed || formatError) return
    setBusy(true)
    setError(null)
    try {
      const device = await api.createDevice({ device_id: trimmed, name: name.trim() })
      setDeviceId('')
      setName('')
      onCreated(device)
    } catch (err) {
      setError(err)
    } finally {
      setBusy(false)
    }
  }

  return (
    <Modal open={open} title="Register a device" onClose={close}>
      <form id="add-device" className="space-y-4" onSubmit={submit} noValidate>
        {error && !apiFieldError ? <Notice>{errorMessage(error)}</Notice> : null}
        <TextField
          label="Device ID"
          placeholder="esp32-hydro-01"
          autoFocus
          required
          value={deviceId}
          onChange={(e) => setDeviceId(e.target.value)}
          error={formatError ?? apiFieldError}
          hint={
            <>
              Must match <code className="rounded bg-canvas px-1">DEVICE_ID</code> in the firmware's{' '}
              <code className="rounded bg-canvas px-1">include/config.h</code>. Telemetry from unregistered IDs is
              ignored.
            </>
          }
        />
        <TextField
          label="Name"
          placeholder="Lettuce tank (optional)"
          value={name}
          onChange={(e) => setName(e.target.value)}
        />
        <div className="flex justify-end gap-2 pt-2">
          <Button variant="secondary" onClick={close}>
            Cancel
          </Button>
          <Button type="submit" loading={busy} disabled={!trimmed || formatError !== null}>
            Register device
          </Button>
        </div>
      </form>
    </Modal>
  )
}
