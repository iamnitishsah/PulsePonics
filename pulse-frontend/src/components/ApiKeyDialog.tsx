import { API_URL } from '../lib/config'
import type { DeviceWithKey } from '../types/api'
import { Button } from './ui/Button'
import { CopyField } from './ui/CopyField'
import { Modal } from './ui/Modal'
import { Notice } from './ui/Notice'

/** Shows a freshly created or rotated API key, which is never shown again. */
export function ApiKeyDialog({ device, onClose }: { device: DeviceWithKey | null; onClose: () => void }) {
  const base = API_URL || window.location.origin
  const curl = device
    ? `curl -X POST ${base}/api/readings \\
  -H "X-Device-Key: ${device.api_key}" \\
  -H "Content-Type: application/json" \\
  -d '{"water_temp_c": 21.5, "ph": 6.1}'`
    : ''

  return (
    <Modal
      open={device !== null}
      title={device ? `${device.name} is ready` : ''}
      onClose={onClose}
      footer={<Button onClick={onClose}>Done</Button>}
    >
      {device ? (
        <div className="space-y-4 text-sm">
          <Notice tone="warning">Copy the API key now. It is stored hashed and cannot be shown again.</Notice>
          <CopyField label="Device API key (HTTP ingestion)" value={device.api_key} />

          <div>
            <p className="font-semibold text-ink">Connect the ESP32 (MQTT)</p>
            <ol className="mt-2 list-decimal space-y-1.5 pl-5 text-ink-soft">
              <li>
                In <code className="rounded bg-canvas px-1">include/config.h</code>, set{' '}
                <code className="rounded bg-canvas px-1">DEVICE_ID</code> to{' '}
                <code className="rounded bg-canvas px-1 font-semibold text-ink">{device.device_id}</code>.
              </li>
              <li>
                In <code className="rounded bg-canvas px-1">include/secrets.h</code>, set{' '}
                <code className="rounded bg-canvas px-1">MQTT_USER</code> /{' '}
                <code className="rounded bg-canvas px-1">MQTT_PASSWORD</code> to the broker's device login (
                <code className="rounded bg-canvas px-1">MQTT_DEVICE_*</code> in the server's .env).
              </li>
              <li>Flash the board. Readings appear here as soon as they arrive.</li>
            </ol>
          </div>

          <CopyField label="Or send a test reading over HTTP" value={curl} multiline />
        </div>
      ) : null}
    </Modal>
  )
}
