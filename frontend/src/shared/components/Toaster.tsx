import * as RadixToast from '@radix-ui/react-toast'
import { X, CheckCircle2, AlertCircle, Info } from 'lucide-react'
import { cn } from '../../lib/utils'
import { useToastStore, type ToastVariant } from '../hooks/useToast'

const VARIANT_CONFIG: Record<ToastVariant, { icon: React.ElementType; className: string; iconClass: string }> = {
  success: {
    icon: CheckCircle2,
    className: 'border-green-500/30 bg-green-500/10',
    iconClass: 'text-green-500',
  },
  error: {
    icon: AlertCircle,
    className: 'border-red-500/30 bg-red-500/10',
    iconClass: 'text-red-500',
  },
  info: {
    icon: Info,
    className: 'border-brand/30 bg-brand/10',
    iconClass: 'text-brand',
  },
}

export function Toaster() {
  const { toasts, dismiss } = useToastStore()

  return (
    <RadixToast.Provider swipeDirection="right">
      {toasts.map(({ id, message, variant }) => {
        const cfg = VARIANT_CONFIG[variant]
        const Icon = cfg.icon
        return (
          <RadixToast.Root
            key={id}
            open
            onOpenChange={(open) => { if (!open) dismiss(id) }}
            className={cn(
              'flex items-start gap-3 rounded-lg border px-4 py-3 shadow-lg',
              'data-[state=open]:animate-in data-[state=closed]:animate-out',
              'data-[state=closed]:fade-out-80 data-[state=open]:fade-in-0',
              'data-[state=closed]:slide-out-to-right-full data-[state=open]:slide-in-from-top-full',
              'bg-surface text-primary text-sm',
              cfg.className,
            )}
          >
            <Icon className={cn('w-4 h-4 shrink-0 mt-0.5', cfg.iconClass)} />
            <RadixToast.Title className="flex-1 leading-snug">{message}</RadixToast.Title>
            <RadixToast.Close
              className="text-secondary hover:text-primary transition-colors shrink-0"
              aria-label="Schließen"
              onClick={() => dismiss(id)}
            >
              <X className="w-3.5 h-3.5" />
            </RadixToast.Close>
          </RadixToast.Root>
        )
      })}
      <RadixToast.Viewport className="fixed top-4 right-4 z-50 flex flex-col gap-2 w-[360px] max-w-[calc(100vw-2rem)]" />
    </RadixToast.Provider>
  )
}
