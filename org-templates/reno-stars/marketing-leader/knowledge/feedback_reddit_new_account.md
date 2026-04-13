---
name: Reddit new accounts have ~24-48h provisioning delay
description: Fresh Reddit accounts can't post media to own profile, edit profile/subreddit settings, or upload avatar/banner until Reddit finishes provisioning the user-subreddit (~24-48h after account creation)
type: feedback
---

When a Reddit account is brand new (less than ~48h old), Reddit's backend hasn't finished provisioning the underlying user-subreddit. This blocks several operations that all return cryptic errors:

**Symptoms:**
- New Reddit shreddit profile settings (`/settings/profile`): "We had some issues saving your changes. Please try again." Console shows "No profile ID for profile settings page".
- Old Reddit subreddit settings (`/user/<name>/about/edit`): HTTP 500 from `/api/site_admin`.
- Posting media to own profile via new Reddit: "Hmm, that community doesn't exist. Try checking the spelling." (even when posting to your own user profile, which is technically `r/u_<name>`).
- Old Reddit submit: hits aggressive reCAPTCHA challenge that automation can't solve.

**Why:** Reddit creates the User Profile properly but the underlying `r/u_<username>` subreddit infrastructure (which holds avatar, banner, settings, profile posts) is provisioned asynchronously. New accounts hit "no profile ID" errors until that completes.

**How to apply:**
- For Reno Stars Reddit account u/Anxious-Owl-9826 (created 2026-04-06): wait until ~2026-04-08 minimum before retrying profile setup or media posts.
- Don't burn time troubleshooting "community doesn't exist" / 500 errors / save failures on a fresh account — they're not bugs in the form, they're the provisioning delay.
- Helpful first replies to other people's posts (text comments, what we did successfully on 2026-04-07) DO work on day-0 accounts. It's only profile/media-post operations that need provisioning.
- After 48h, also have the user verify the account email — unverified accounts get extra friction.
