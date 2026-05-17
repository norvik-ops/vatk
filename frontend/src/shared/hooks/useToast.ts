import { useState, useCallback } from 'react'

export type ToastVariant = 'success' | 'error' | 'info'

export interface ToastMessage {
  id: number
  message: string
  variant: ToastVariant
}

let _addToast: ((msg: string, variant: ToastVariant) => void) | null = null
let _counter = 0

/**
 * Global toast store — used by the Toaster component and the toast() helper.
 */
export function useToastStore() {
  const [toasts, setToasts] = useState<ToastMessage[]>([])

  const addToast = useCallback((message: string, variant: ToastVariant = 'info') => {
    const id = ++_counter
    setToasts((prev) => [...prev, { id, message, variant }])
    setTimeout(() => {
      setToasts((prev) => prev.filter((t) => t.id !== id))
    }, 4000)
  }, [])

  // Register the global handler when this store mounts inside <Toaster />
  _addToast = addToast

  const dismiss = useCallback((id: number) => {
    setToasts((prev) => prev.filter((t) => t.id !== id))
  }, [])

  return { toasts, addToast, dismiss }
}

/**
 * Imperative toast() helper — can be called from anywhere.
 * The Toaster component must be mounted for this to work.
 */
export function toast(message: string, variant: ToastVariant = 'info') {
  if (_addToast) {
    _addToast(message, variant)
  }
}
