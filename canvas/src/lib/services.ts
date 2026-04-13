import type { ServiceConfig, SecretGroup } from '@/types/secrets';

/**
 * Static service registry. Each known provider maps to its display
 * properties, expected key names, and whether test-connection is supported.
 *
 * Keys not matching any known service fall into the "custom" catch-all.
 */
export const SERVICES: Record<SecretGroup, ServiceConfig> = {
  github: {
    label: 'GitHub',
    icon: 'github',
    keyNames: ['GITHUB_TOKEN'],
    docsUrl: 'https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens',
    testSupported: true,
  },
  anthropic: {
    label: 'Anthropic',
    icon: 'anthropic',
    keyNames: ['ANTHROPIC_API_KEY'],
    docsUrl: 'https://docs.anthropic.com/en/api/getting-started',
    testSupported: true,
  },
  openrouter: {
    label: 'OpenRouter',
    icon: 'openrouter',
    keyNames: ['OPENROUTER_API_KEY'],
    docsUrl: 'https://openrouter.ai/docs/api-keys',
    testSupported: true,
  },
  custom: {
    label: 'Other',
    icon: 'key',
    keyNames: [],
    docsUrl: '',
    testSupported: false,
  },
};

/** Ordered list of groups for consistent rendering. */
export const SERVICE_GROUP_ORDER: SecretGroup[] = [
  'github',
  'anthropic',
  'openrouter',
  'custom',
];

/** Get default key name when a service is selected in the Add form. */
export function getDefaultKeyName(group: SecretGroup): string {
  return SERVICES[group].keyNames[0] ?? '';
}
