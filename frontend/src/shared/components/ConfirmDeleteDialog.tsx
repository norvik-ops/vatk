import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '../../components/ui/dialog'
import { Button } from '../../components/ui/button'
import { Input } from '../../components/ui/input'
import { Label } from '../../components/ui/label'

interface ConfirmDeleteDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  /** The exact name the user must type to confirm. */
  resourceName: string
  /** Used in the title, e.g. "Asset" → "Asset löschen?" */
  resourceType: string
  onConfirm: () => void
  isPending?: boolean
}

export function ConfirmDeleteDialog({
  open,
  onOpenChange,
  resourceName,
  resourceType,
  onConfirm,
  isPending = false,
}: ConfirmDeleteDialogProps) {
  const [inputValue, setInputValue] = useState('')

  function handleOpenChange(next: boolean) {
    if (!next) setInputValue('')
    onOpenChange(next)
  }

  function handleConfirm() {
    if (inputValue !== resourceName) return
    onConfirm()
    setInputValue('')
  }

  const canDelete = inputValue === resourceName

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{resourceType} löschen?</DialogTitle>
        </DialogHeader>
        <div className="space-y-4 py-2">
          <p className="text-sm text-secondary">
            Diese Aktion ist nicht rückgängig zu machen. Gib den Namen ein um zu bestätigen:
          </p>
          <div className="space-y-1.5">
            <Label htmlFor="confirm-delete-input">
              <span className="font-mono text-xs bg-surface2 px-1.5 py-0.5 rounded border border-border">
                {resourceName}
              </span>
            </Label>
            <Input
              id="confirm-delete-input"
              value={inputValue}
              onChange={(e) => { setInputValue(e.target.value); }}
              placeholder={resourceName}
              autoComplete="off"
            />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => { handleOpenChange(false); }} disabled={isPending}>
            Abbrechen
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={!canDelete || isPending}
          >
            {isPending ? 'Wird gelöscht…' : 'Löschen'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
