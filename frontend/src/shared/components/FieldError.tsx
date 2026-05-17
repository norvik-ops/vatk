interface FieldErrorProps {
  error: string | null
}

export function FieldError({ error }: FieldErrorProps) {
  return (
    <div
      role="alert"
      aria-live="polite"
      className={`overflow-hidden transition-all duration-200 ${
        error ? 'max-h-10 opacity-100 mt-1' : 'max-h-0 opacity-0'
      }`}
    >
      {error && (
        <p className="text-xs text-red-600 dark:text-red-400">{error}</p>
      )}
    </div>
  )
}
