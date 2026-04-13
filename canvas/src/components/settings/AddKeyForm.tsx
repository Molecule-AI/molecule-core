'use client';
import { useState, useCallback, useEffect, useRef } from 'react';
import type { SecretGroup } from '@/types/secrets';
import { useSecretsStore } from '@/stores/secrets-store';
import { KeyValueField } from '@/components/ui/KeyValueField';
import { ValidationHint } from '@/components/ui/ValidationHint';
import { TestConnectionButton } from '@/components/ui/TestConnectionButton';
import {
  validateSecretValue,
  isValidKeyName,
  inferGroup,
} from '@/lib/validation/secret-formats';
import { SERVICES, SERVICE_GROUP_ORDER, getDefaultKeyName } from '@/lib/services';

const VALIDATION_DEBOUNCE_MS = 400;

interface AddKeyFormProps {
  workspaceId: string;
  existingNames: string[];
  onCancel: () => void;
}

/**
 * Inline-expanding form for adding a new API key.
 *
 * Flow (from spec §4.2):
 *   Form Open → select service → key name auto-fills → type value →
 *   optional Test Connection → Save
 */
export function AddKeyForm({
  workspaceId,
  existingNames,
  onCancel,
}: AddKeyFormProps) {
  const createSecret = useSecretsStore((s) => s.createSecret);

  const [selectedGroup, setSelectedGroup] = useState<SecretGroup>('github');
  const [keyName, setKeyName] = useState(getDefaultKeyName('github'));
  const [value, setValue] = useState('');
  const [validationError, setValidationError] = useState<string | null>(null);
  const [keyNameError, setKeyNameError] = useState<string | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const debounceRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const service = SERVICES[selectedGroup];

  // Auto-fill key name when service changes
  const handleServiceChange = useCallback(
    (group: SecretGroup) => {
      setSelectedGroup(group);
      const defaultName = getDefaultKeyName(group);
      if (defaultName) {
        setKeyName(defaultName);
      }
      // Reset validation
      setValidationError(null);
      setKeyNameError(null);
      setSaveError(null);
    },
    [],
  );

  // Validate key name
  useEffect(() => {
    if (!keyName) {
      setKeyNameError(null);
      return;
    }
    if (!isValidKeyName(keyName)) {
      setKeyNameError('Key name must be UPPER_SNAKE_CASE');
      return;
    }
    if (existingNames.includes(keyName)) {
      setKeyNameError('A key named ' + keyName + ' already exists. Edit it instead.');
      return;
    }
    setKeyNameError(null);
  }, [keyName, existingNames]);

  // Debounced value validation
  useEffect(() => {
    if (!value) {
      setValidationError(null);
      return;
    }
    clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setValidationError(validateSecretValue(value, selectedGroup));
    }, VALIDATION_DEBOUNCE_MS);
    return () => clearTimeout(debounceRef.current);
  }, [value, selectedGroup]);

  const handleSave = useCallback(async () => {
    // Final validation pass
    if (!isValidKeyName(keyName)) {
      setKeyNameError('Key name must be UPPER_SNAKE_CASE');
      return;
    }
    const valErr = validateSecretValue(value, selectedGroup);
    if (valErr) {
      setValidationError(valErr);
      return;
    }

    setIsSaving(true);
    setSaveError(null);
    try {
      await createSecret(workspaceId, keyName, value);
      // Form auto-closes via store (isAddFormOpen set to false)
    } catch (e) {
      const message = e instanceof Error ? e.message : 'Failed to save. Check your connection and try again.';
      setSaveError(message);
    } finally {
      setIsSaving(false);
    }
  }, [keyName, value, selectedGroup, createSecret, workspaceId]);

  const canSave = keyName && value && !keyNameError && !validationError && !isSaving;

  return (
    <div className="add-key-form">
      <div className="add-key-form__header">Add New Key</div>

      {/* Service selector */}
      <label className="add-key-form__label">
        Service
        <select
          value={selectedGroup}
          onChange={(e) => handleServiceChange(e.target.value as SecretGroup)}
          disabled={isSaving}
          className="add-key-form__select"
        >
          {SERVICE_GROUP_ORDER.map((group) => (
            <option key={group} value={group}>
              {SERVICES[group].label}
            </option>
          ))}
        </select>
      </label>

      {/* Key name */}
      <label className="add-key-form__label">
        Key name
        <input
          type="text"
          value={keyName}
          onChange={(e) => setKeyName(e.target.value.toUpperCase())}
          disabled={isSaving}
          placeholder="MY_API_KEY"
          className="add-key-form__input"
          autoComplete="off"
          spellCheck={false}
        />
      </label>
      {keyNameError && (
        <ValidationHint error={keyNameError} />
      )}

      {/* Key value */}
      <label className="add-key-form__label">
        Value
      </label>
      <KeyValueField
        value={value}
        onChange={setValue}
        disabled={isSaving}
        aria-label={`Value for ${keyName || 'new key'}`}
      />
      <ValidationHint
        error={validationError}
        showValid={!validationError && value.length > 0}
      />

      {/* Test connection (only for supported services) */}
      {service.testSupported && value && !validationError && (
        <TestConnectionButton
          provider={selectedGroup}
          secretValue={value}
        />
      )}

      {/* Save error */}
      {saveError && (
        <div className="add-key-form__error" role="alert">
          {saveError}
        </div>
      )}

      {/* Actions */}
      <div className="add-key-form__actions">
        <button
          type="button"
          onClick={onCancel}
          disabled={isSaving}
          className="add-key-form__cancel-btn"
        >
          Cancel
        </button>
        <button
          type="button"
          onClick={handleSave}
          disabled={!canSave}
          className="add-key-form__save-btn"
        >
          {isSaving ? 'Saving…' : 'Save key'}
        </button>
      </div>
    </div>
  );
}
