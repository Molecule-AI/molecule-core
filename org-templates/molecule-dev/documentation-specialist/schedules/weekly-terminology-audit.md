Weekly audit of documentation freshness and terminology consistency.

1. STALE PAGE DETECTION:
   cd /workspace/docs && for f in content/docs/*.mdx; do
     age=$(git log -1 --format='%cr' -- "$f")
     echo "$age :: $f"
   done | sort -r
   Flag any page not touched in 30+ days that covers a
   fast-moving surface (handlers, plugins, templates).

2. TERMINOLOGY CONSISTENCY:
   grep -rEi "workspace|agent|cron|schedule|plugin|channel|template" \
     content/docs/*.mdx | grep -oE "\b(workspace|workspaces|Agent|agent|cron job|schedule|plugin|channel|template)\b" | \
     sort | uniq -c | sort -rn
   Each concept should have ONE canonical capitalisation and
   plural form. Open a PR fixing inconsistencies.

3. LINK ROT:
   grep -rE "\[.*\]\(http[^)]+\)" content/docs/*.mdx | \
   awk -F'[()]' '{print $2}' | sort -u | \
   while read url; do
     curl -sIo /dev/null -w "%{http_code} $url\n" "$url"
   done | grep -v "^200 "
   Report any non-200 to PM.

4. ROUTING + MEMORY:
   Same audit_summary contract as the daily cron.
   Save findings to memory key 'docs-weekly-audit'.
