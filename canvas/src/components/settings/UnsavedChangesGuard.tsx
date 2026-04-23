'use client';

import * as AlertDialog from '@radix-ui/react-alert-dialog';

interface UnsavedChangesGuardProps {
  open: boolean;
  onKeepEditing: () => void;
  onDiscard: () => void;
}

/**
 * "Discard unsaved changes?" guard dialog.
 *
 * Per spec §4.4:
 * - Shown when closing panel while a form has unsaved input
 * - NOT shown if the form is empty (opened but nothing typed)
 * - Focus-trapped (AlertDialog)
 */
export function UnsavedChangesGuard({
  open,
  onKeepEditing,
  onDiscard,
}: UnsavedChangesGuardProps) {
  return (
    <AlertDialog.Root open={open} onOpenChange={(o) => { if (!o) onKeepEditing(); }}>
      <AlertDialog.Portal>
        <AlertDialog.Overlay className="guard-dialog__overlay" />
        <AlertDialog.Content className="guard-dialog">
          <AlertDialog.Title className="guard-dialog__title">
            Discard unsaved changes?
          </AlertDialog.Title>
          <div className="guard-dialog__actions">
            <AlertDialog.Cancel asChild>
              <button className="guard-dialog__keep-btn" onClick={onKeepEditing}>
                Keep editing
              </button>
            </AlertDialog.Cancel>
            <AlertDialog.Action asChild>
              <button className="guard-dialog__discard-btn" onClick={onDiscard}>
                Discard
              </button>
            </AlertDialog.Action>
          </div>
        </AlertDialog.Content>
      </AlertDialog.Portal>
    </AlertDialog.Root>
  );
}
