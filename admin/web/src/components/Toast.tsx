import { useEffect, useState } from 'react'

export function ToastContainer({
  message,
  type,
  onDone,
}: {
  message: string
  type: 'success' | 'error'
  onDone: () => void
}) {
  useEffect(() => {
    const t = setTimeout(onDone, 4000)
    return () => clearTimeout(t)
  }, [onDone])

  if (!message) return null

  return (
    <div className="toast-container">
      <div className={`toast ${type}`}>{message}</div>
    </div>
  )
}

export function useToast() {
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' } | null>(
    null,
  )
  const show = (message: string, type: 'success' | 'error' = 'success') => setToast({ message, type })
  const clear = () => setToast(null)
  return { toast, show, clear }
}
