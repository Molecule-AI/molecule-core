DELETE FROM platform_instructions
WHERE scope = 'global' AND title IN (
  'Verify before claiming',
  'CRITICAL/P0/URGENT requires raw evidence',
  'Circuit breaker — stop the retry cascade',
  'Do not invent phases, deadlines, or features',
  'Token expiry is a known issue, not a P0',
  'Slack noise discipline',
  'Identity tag every external comment',
  'Staging-first workflow, no exceptions'
);
