"""Slack webhook client for posting brand mentions and daily digest."""

import os
import logging
import requests

logger = logging.getLogger(__name__)

# Competitor names that auto-trigger @here alert
COMPETITOR_NAMES = [
    "openai", "langchain", "langgraph", "autogen", "crewai", "crew ai",
    "llamaindex", "dify", "flowise", "n8n", "zapier", "make.com",
]

# Engagement threshold above which @here is triggered
AT_HERE_ENGAGEMENT_THRESHOLD = 10


class SlackClient:
    """Posts brand mention alerts and daily digests to a Slack webhook.

    Webhook URL from SLACK_WEBHOOK_URL env var.
    """

    def __init__(self):
        self.webhook_url = os.environ.get("SLACK_WEBHOOK_URL")
        if not self.webhook_url:
            raise EnvironmentError("Missing required environment variable: SLACK_WEBHOOK_URL")

    # ------------------------------------------------------------------
    # Internal helpers
    # ------------------------------------------------------------------

    def _engagement_score(self, tweet):
        """Sum of likes + retweets + replies."""
        metrics = tweet.get("public_metrics", {})
        return (
            metrics.get("like_count", 0)
            + metrics.get("retweet_count", 0)
            + metrics.get("reply_count", 0)
        )

    def _escape_mrkdwn(self, text: str) -> str:
        """Escape Slack mrkdwn special characters in untrusted content."""
        return text.replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")

    def _should_at_here(self, tweet):
        """Return True if the tweet warrants an @here ping."""
        if self._engagement_score(tweet) > AT_HERE_ENGAGEMENT_THRESHOLD:
            return True
        text = tweet.get("text", "").lower()
        return any(comp in text for comp in COMPETITOR_NAMES)

    def _format_tweet_block(self, tweet):
        """Format a single tweet as a Slack mrkdwn string."""
        tweet_id = tweet.get("id", "")
        author_id = tweet.get("author_id", "unknown")
        text = tweet.get("text", "").replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
        created_at = tweet.get("created_at", "")
        metrics = tweet.get("public_metrics", {})
        url = f"https://twitter.com/i/web/status/{tweet_id}"

        return (
            f"*New mention* — <{url}|view>\n"
            f">{text}\n"
            f"Author: `{author_id}` | "
            f"❤️ {metrics.get('like_count', 0)}  "
            f"🔁 {metrics.get('retweet_count', 0)}  "
            f"💬 {metrics.get('reply_count', 0)}\n"
            f"_Posted: {created_at}_"
        )

    # ------------------------------------------------------------------
    # Public API
    # ------------------------------------------------------------------

    def post_mentions(self, tweets):
        """Bundle and post new brand mentions to Slack.

        Multiple tweets are sent in a single webhook payload, not one per tweet.

        Args:
            tweets: List of tweet dicts from XClient.search_recent().

        Returns:
            None. No-ops on empty list.

        Raises:
            requests.HTTPError: On non-2xx Slack response.
        """
        if not tweets:
            return

        has_at_here = any(self._should_at_here(t) for t in tweets)

        blocks = []
        if has_at_here:
            blocks.append(
                {"type": "section", "text": {"type": "mrkdwn", "text": "<!here>"}}
            )

        count = len(tweets)
        header = f"*{count} new Molecule AI mention{'s' if count > 1 else ''}* in #brand-monitoring"
        blocks.append({"type": "section", "text": {"type": "mrkdwn", "text": header}})
        blocks.append({"type": "divider"})

        for tweet in tweets:
            blocks.append(
                {"type": "section", "text": {"type": "mrkdwn", "text": self._format_tweet_block(tweet)}}
            )
            blocks.append({"type": "divider"})

        payload = {"blocks": blocks}
        logger.info("Posting %d mention(s) to Slack (at_here=%s)", count, has_at_here)
        response = requests.post(self.webhook_url, json=payload, timeout=15)
        response.raise_for_status()

    def post_digest(self, summary):
        """Post the daily 20:00 UTC mention digest to Slack.

        Args:
            summary: Dict with keys:
                count (int): total mentions today
                top_tweets (list, optional): list of high-engagement tweet dicts

        Raises:
            requests.HTTPError: On non-2xx Slack response.
        """
        count = summary.get("count", 0)
        top_tweets = summary.get("top_tweets", [])

        lines = [
            "*📊 Daily Digest — Molecule AI Brand Mentions*",
            f"Total mentions today: *{count}*",
        ]

        if top_tweets:
            lines.append("\n*Top engagements:*")
            for tweet in top_tweets[:3]:
                snippet = self._escape_mrkdwn(tweet.get("text", "")[:120])
                score = self._engagement_score(tweet)
                tweet_id = tweet.get("id", "")
                url = f"https://twitter.com/i/web/status/{tweet_id}"
                lines.append(f"• <{url}|{snippet}…>  _(score: {score})_")

        payload = {"text": "\n".join(lines)}
        logger.info("Posting daily digest to Slack (count=%d)", count)
        response = requests.post(self.webhook_url, json=payload, timeout=15)
        response.raise_for_status()
