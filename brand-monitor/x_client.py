"""X API v2 thin client for brand mention search."""

import os
import logging
import requests

logger = logging.getLogger(__name__)

SEARCH_URL = "https://api.twitter.com/2/tweets/search/recent"

# Verbatim from issue #549 — drug-discovery SEO noise suppressed at query level
SEARCH_QUERY = (
    '("Molecule AI" OR "@moleculeai") '
    '(agent OR workflow OR orchestrat OR "multi-agent" OR developer OR SDK OR API OR "agent platform") '
    '-moleculeai.com -molecule.ai -"drug discovery" -pharmaceutical -CRISPR -oncology '
    '-is:retweet lang:en'
)

TWEET_FIELDS = "author_id,created_at,public_metrics,entities"


class XClient:
    """Thin wrapper around X API v2 recent-search endpoint.

    Auth: Bearer token from X_BEARER_TOKEN env var.
    """

    def __init__(self):
        self.bearer_token = os.environ.get("X_BEARER_TOKEN")
        if not self.bearer_token:
            raise EnvironmentError("Missing required environment variable: X_BEARER_TOKEN")

    def search_recent(self, since_id=None, start_time=None, max_results=100):
        """Search recent tweets matching SEARCH_QUERY.

        Args:
            since_id: Only return tweets newer than this tweet ID.
            start_time: ISO 8601 datetime string; only return tweets after this time.
            max_results: Max tweets per request (10–100).

        Returns:
            List of tweet dicts (newest first), empty list if none found.

        Raises:
            requests.HTTPError: On non-2xx API response.
        """
        headers = {"Authorization": f"Bearer {self.bearer_token}"}
        params = {
            "query": SEARCH_QUERY,
            "tweet.fields": TWEET_FIELDS,
            "max_results": max_results,
        }
        if since_id:
            params["since_id"] = since_id
        if start_time:
            params["start_time"] = start_time

        logger.debug("Searching X API: since_id=%s start_time=%s", since_id, start_time)
        response = requests.get(SEARCH_URL, headers=headers, params=params, timeout=30)
        response.raise_for_status()

        data = response.json()
        tweets = data.get("data", [])
        logger.info("X API returned %d tweet(s)", len(tweets))
        return tweets
