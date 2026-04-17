"""Full test suite for brand-monitor modules.

Run:
    pytest test_monitor.py -v --cov=. --cov-report=term-missing --cov-fail-under=100

All HTTP calls are mocked — no live API calls, no credentials needed.
"""

import json
import os
from datetime import datetime, timedelta, timezone
from unittest.mock import MagicMock, call, patch

import pytest
import requests

# ---------------------------------------------------------------------------
# Shared fixtures / constants
# ---------------------------------------------------------------------------

BASE_ENV = {
    "X_BEARER_TOKEN": "test-bearer-token",
    "X_API_KEY": "test-api-key",
    "X_API_SECRET": "test-api-secret",
    "SLACK_WEBHOOK_URL": "https://hooks.slack.com/services/TEST",
}

SAMPLE_TWEET = {
    "id": "1111111111",
    "text": "Really excited about Molecule AI's agent platform — great SDK!",
    "author_id": "9876543210",
    "created_at": "2024-01-01T12:00:00Z",
    "public_metrics": {
        "like_count": 3,
        "retweet_count": 1,
        "reply_count": 2,
    },
}

SAMPLE_TWEET_HIGH_ENGAGEMENT = {
    "id": "2222222222",
    "text": "Molecule AI multi-agent workflow is incredible",
    "author_id": "1111111111",
    "created_at": "2024-01-01T13:00:00Z",
    "public_metrics": {
        "like_count": 50,
        "retweet_count": 20,
        "reply_count": 15,
    },
}

SAMPLE_TWEET_COMPETITOR = {
    "id": "3333333333",
    "text": "Comparing Molecule AI with langchain for our orchestration workflow",
    "author_id": "2222222222",
    "created_at": "2024-01-01T14:00:00Z",
    "public_metrics": {
        "like_count": 0,
        "retweet_count": 0,
        "reply_count": 0,
    },
}


# ===========================================================================
# x_client tests
# ===========================================================================


class TestXClient:

    def test_init_missing_token_raises(self):
        from x_client import XClient

        with patch.dict(os.environ, {}, clear=True):
            with pytest.raises(EnvironmentError, match="X_BEARER_TOKEN"):
                XClient()

    def test_init_success(self):
        from x_client import XClient

        with patch.dict(os.environ, {"X_BEARER_TOKEN": "my-token"}):
            client = XClient()
        assert client.bearer_token == "my-token"

    def _make_client(self):
        from x_client import XClient

        with patch.dict(os.environ, {"X_BEARER_TOKEN": "tok"}):
            return XClient()

    def test_search_recent_returns_tweets(self):
        from x_client import SEARCH_QUERY, SEARCH_URL

        client = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None
        mock_resp.json.return_value = {"data": [SAMPLE_TWEET]}

        with patch("x_client.requests.get", return_value=mock_resp) as mock_get:
            result = client.search_recent()

        assert result == [SAMPLE_TWEET]
        # Verify URL, auth header and query string
        args, kwargs = mock_get.call_args
        assert args[0] == SEARCH_URL
        assert kwargs["headers"]["Authorization"] == "Bearer tok"
        assert kwargs["params"]["query"] == SEARCH_QUERY

    def test_search_recent_no_data_key_returns_empty_list(self):
        client = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None
        mock_resp.json.return_value = {"meta": {"result_count": 0}}

        with patch("x_client.requests.get", return_value=mock_resp):
            result = client.search_recent()

        assert result == []

    def test_search_recent_with_since_id_adds_param(self):
        client = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None
        mock_resp.json.return_value = {"data": [SAMPLE_TWEET]}

        with patch("x_client.requests.get", return_value=mock_resp) as mock_get:
            client.search_recent(since_id="9999")

        params = mock_get.call_args.kwargs["params"]
        assert params["since_id"] == "9999"

    def test_search_recent_with_start_time_adds_param(self):
        client = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None
        mock_resp.json.return_value = {"data": []}

        with patch("x_client.requests.get", return_value=mock_resp) as mock_get:
            client.search_recent(start_time="2024-01-01T00:00:00Z")

        params = mock_get.call_args.kwargs["params"]
        assert params["start_time"] == "2024-01-01T00:00:00Z"

    def test_search_recent_no_since_id_no_start_time_omits_params(self):
        """Neither since_id nor start_time in params when not provided."""
        client = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None
        mock_resp.json.return_value = {"data": []}

        with patch("x_client.requests.get", return_value=mock_resp) as mock_get:
            client.search_recent()

        params = mock_get.call_args.kwargs["params"]
        assert "since_id" not in params
        assert "start_time" not in params

    def test_search_recent_http_error_propagates(self):
        client = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.side_effect = requests.HTTPError("403 Forbidden")

        with patch("x_client.requests.get", return_value=mock_resp):
            with pytest.raises(requests.HTTPError):
                client.search_recent()


# ===========================================================================
# slack_client tests
# ===========================================================================


class TestSlackClient:

    def _make_client(self):
        from slack_client import SlackClient

        with patch.dict(os.environ, {"SLACK_WEBHOOK_URL": "https://hooks.slack.com/test"}):
            return SlackClient()

    def test_init_missing_webhook_raises(self):
        from slack_client import SlackClient

        with patch.dict(os.environ, {}, clear=True):
            with pytest.raises(EnvironmentError, match="SLACK_WEBHOOK_URL"):
                SlackClient()

    def test_init_success(self):
        c = self._make_client()
        assert c.webhook_url == "https://hooks.slack.com/test"

    def test_engagement_score_sums_correctly(self):
        c = self._make_client()
        tweet = {"public_metrics": {"like_count": 5, "retweet_count": 3, "reply_count": 2}}
        assert c._engagement_score(tweet) == 10

    def test_engagement_score_missing_metrics_returns_zero(self):
        c = self._make_client()
        assert c._engagement_score({}) == 0

    def test_should_at_here_high_engagement_returns_true(self):
        c = self._make_client()
        assert c._should_at_here(SAMPLE_TWEET_HIGH_ENGAGEMENT) is True

    def test_should_at_here_competitor_name_returns_true(self):
        c = self._make_client()
        # SAMPLE_TWEET_COMPETITOR contains "langchain" — engagement is 0
        assert c._should_at_here(SAMPLE_TWEET_COMPETITOR) is True

    def test_should_at_here_normal_tweet_returns_false(self):
        c = self._make_client()
        # SAMPLE_TWEET: engagement=6 (<=10), no competitor
        assert c._should_at_here(SAMPLE_TWEET) is False

    def test_post_mentions_empty_list_is_noop(self):
        c = self._make_client()
        with patch("slack_client.requests.post") as mock_post:
            c.post_mentions([])
        mock_post.assert_not_called()

    def test_post_mentions_single_tweet_no_at_here(self):
        c = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None

        with patch("slack_client.requests.post", return_value=mock_resp) as mock_post:
            c.post_mentions([SAMPLE_TWEET])

        mock_post.assert_called_once()
        payload = mock_post.call_args.kwargs["json"]
        section_texts = [
            b["text"]["text"]
            for b in payload["blocks"]
            if b.get("type") == "section"
        ]
        # No @here for normal engagement tweet
        assert not any("<!here>" in t for t in section_texts)
        # Header mentions "1 new … mention"
        assert any("1 new" in t for t in section_texts)

    def test_post_mentions_multiple_tweets_with_at_here(self):
        """High-engagement tweet triggers @here; both tweets appear in payload."""
        c = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None

        with patch("slack_client.requests.post", return_value=mock_resp) as mock_post:
            c.post_mentions([SAMPLE_TWEET_HIGH_ENGAGEMENT, SAMPLE_TWEET])

        payload = mock_post.call_args.kwargs["json"]
        section_texts = [
            b["text"]["text"]
            for b in payload["blocks"]
            if b.get("type") == "section"
        ]
        assert any("<!here>" in t for t in section_texts)
        assert any("2 new" in t for t in section_texts)

    def test_post_mentions_html_escaping_in_tweet_text(self):
        """< > & in tweet text are escaped to prevent Slack mrkdwn injection."""
        c = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None
        tweet = {**SAMPLE_TWEET, "text": "X < Y & Z > W"}

        with patch("slack_client.requests.post", return_value=mock_resp) as mock_post:
            c.post_mentions([tweet])

        raw = str(mock_post.call_args.kwargs["json"])
        assert "&lt;" in raw
        assert "&gt;" in raw
        assert "&amp;" in raw

    def test_post_mentions_http_error_propagates(self):
        c = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.side_effect = requests.HTTPError("500")

        with patch("slack_client.requests.post", return_value=mock_resp):
            with pytest.raises(requests.HTTPError):
                c.post_mentions([SAMPLE_TWEET])

    def test_post_digest_count_only_no_top_tweets(self):
        c = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None

        with patch("slack_client.requests.post", return_value=mock_resp) as mock_post:
            c.post_digest({"count": 42})

        text = mock_post.call_args.kwargs["json"]["text"]
        assert "42" in text
        assert "Top engagements" not in text

    def test_post_digest_with_top_tweets_included(self):
        c = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.return_value = None

        with patch("slack_client.requests.post", return_value=mock_resp) as mock_post:
            c.post_digest({"count": 10, "top_tweets": [SAMPLE_TWEET_HIGH_ENGAGEMENT, SAMPLE_TWEET]})

        text = mock_post.call_args.kwargs["json"]["text"]
        assert "Top engagements" in text

    def test_post_digest_http_error_propagates(self):
        c = self._make_client()
        mock_resp = MagicMock()
        mock_resp.raise_for_status.side_effect = requests.HTTPError("500")

        with patch("slack_client.requests.post", return_value=mock_resp):
            with pytest.raises(requests.HTTPError):
                c.post_digest({"count": 1})


# ===========================================================================
# surge tests
# ===========================================================================


class TestSurgeState:

    def _make_surge(self, tmp_path):
        from surge import SurgeState

        return SurgeState(state_file=str(tmp_path / ".surge_state.json"))

    def test_init_default_state_file(self):
        from surge import DEFAULT_SURGE_FILE, SurgeState

        s = SurgeState()
        assert s.state_file == DEFAULT_SURGE_FILE

    def test_init_custom_state_file(self, tmp_path):
        s = self._make_surge(tmp_path)
        assert ".surge_state.json" in s.state_file

    def test_enable_writes_state_file_with_correct_fields(self, tmp_path):
        s = self._make_surge(tmp_path)
        s.enable(duration_hours=3)
        state = json.loads(open(s.state_file).read())
        assert state["active"] is True
        assert state["duration_hours"] == 3
        assert "expires_at" in state
        assert "enabled_at" in state

    def test_enable_default_duration(self, tmp_path):
        from surge import DEFAULT_SURGE_DURATION_HOURS

        s = self._make_surge(tmp_path)
        s.enable()
        state = json.loads(open(s.state_file).read())
        assert state["duration_hours"] == DEFAULT_SURGE_DURATION_HOURS

    def test_disable_removes_file(self, tmp_path):
        s = self._make_surge(tmp_path)
        s.enable()
        assert os.path.exists(s.state_file)
        s.disable()
        assert not os.path.exists(s.state_file)

    def test_disable_no_file_does_not_raise(self, tmp_path):
        s = self._make_surge(tmp_path)
        # File doesn't exist — should be silent
        s.disable()

    def test_is_active_no_file_returns_false(self, tmp_path):
        s = self._make_surge(tmp_path)
        assert s.is_active() is False

    def test_is_active_not_expired_returns_true(self, tmp_path):
        s = self._make_surge(tmp_path)
        s.enable(duration_hours=6)
        assert s.is_active() is True

    def test_is_active_expired_auto_disables_returns_false(self, tmp_path):
        s = self._make_surge(tmp_path)
        # Write an already-expired state
        past = (datetime.now(timezone.utc) - timedelta(hours=1)).isoformat()
        json.dump({"active": True, "expires_at": past, "duration_hours": 1}, open(s.state_file, "w"))
        assert s.is_active() is False
        assert not os.path.exists(s.state_file)

    def test_check_expiry_returns_true_when_active(self, tmp_path):
        s = self._make_surge(tmp_path)
        s.enable(duration_hours=6)
        assert s.check_expiry() is True

    def test_check_expiry_returns_false_when_expired(self, tmp_path):
        s = self._make_surge(tmp_path)
        past = (datetime.now(timezone.utc) - timedelta(hours=1)).isoformat()
        json.dump({"active": True, "expires_at": past, "duration_hours": 1}, open(s.state_file, "w"))
        assert s.check_expiry() is False

    def test_get_interval_surge_active_returns_surge_interval(self, tmp_path):
        s = self._make_surge(tmp_path)
        s.enable(duration_hours=6)
        assert s.get_interval(1800, 900) == 900

    def test_get_interval_surge_inactive_returns_normal_interval(self, tmp_path):
        s = self._make_surge(tmp_path)
        assert s.get_interval(1800, 900) == 1800


# ===========================================================================
# monitor — validate_env tests
# ===========================================================================


class TestValidateEnv:

    def test_all_vars_present_passes(self):
        from monitor import validate_env

        with patch.dict(os.environ, BASE_ENV, clear=False):
            validate_env()  # must not raise

    def test_single_missing_var_raises_with_name(self):
        from monitor import validate_env

        env = {k: v for k, v in BASE_ENV.items() if k != "X_BEARER_TOKEN"}
        with patch.dict(os.environ, env, clear=True):
            with pytest.raises(EnvironmentError, match="X_BEARER_TOKEN"):
                validate_env()

    def test_multiple_missing_vars_raises_with_all_names(self):
        from monitor import validate_env

        with patch.dict(os.environ, {}, clear=True):
            with pytest.raises(EnvironmentError) as exc_info:
                validate_env()
        msg = str(exc_info.value)
        assert "X_BEARER_TOKEN" in msg
        assert "SLACK_WEBHOOK_URL" in msg


# ===========================================================================
# monitor — enable_surge_mode tests
# ===========================================================================


class TestEnableSurgeMode:

    def test_default_duration_uses_env_default(self, tmp_path):
        from monitor import SURGE_DURATION_HOURS, enable_surge_mode

        sf = str(tmp_path / ".surge.json")
        enable_surge_mode(state_file=sf)
        state = json.loads(open(sf).read())
        assert state["duration_hours"] == SURGE_DURATION_HOURS

    def test_custom_duration_overrides_default(self, tmp_path):
        from monitor import enable_surge_mode

        sf = str(tmp_path / ".surge.json")
        enable_surge_mode(duration_hours=12, state_file=sf)
        state = json.loads(open(sf).read())
        assert state["duration_hours"] == 12

    def test_no_state_file_override_uses_default_path(self):
        """When state_file=None, SurgeState() is constructed with no kwargs."""
        from monitor import enable_surge_mode

        with patch("monitor.SurgeState") as MockSurge:
            mock_instance = MagicMock()
            MockSurge.return_value = mock_instance
            enable_surge_mode(duration_hours=3)

        MockSurge.assert_called_once_with()
        mock_instance.enable.assert_called_once_with(3)


# ===========================================================================
# monitor — Monitor class tests
# ===========================================================================


class TestMonitor:
    """Tests for the Monitor class."""

    # ------------------------------------------------------------------
    # Constructor helpers
    # ------------------------------------------------------------------

    def _make_monitor(self, tmp_path, state_data=None):
        """Build a Monitor with temp files and mocked HTTP clients."""
        from monitor import Monitor

        state_file = str(tmp_path / "monitor_state.json")
        surge_file = str(tmp_path / "surge_state.json")

        if state_data is not None:
            json.dump(state_data, open(state_file, "w"))

        with patch.dict(os.environ, BASE_ENV, clear=False):
            with patch("monitor.XClient"), patch("monitor.SlackClient"):
                m = Monitor(state_file=state_file, surge_state_file=surge_file)
        return m

    # ------------------------------------------------------------------
    # __init__
    # ------------------------------------------------------------------

    def test_init_success_with_empty_state(self, tmp_path):
        m = self._make_monitor(tmp_path)
        assert m.state == {}

    def test_init_loads_existing_state_file(self, tmp_path):
        m = self._make_monitor(tmp_path, state_data={"since_id": "abc"})
        assert m.state["since_id"] == "abc"

    def test_init_missing_env_raises(self, tmp_path):
        from monitor import Monitor

        sf = str(tmp_path / "st.json")
        with patch.dict(os.environ, {}, clear=True):
            with pytest.raises(EnvironmentError):
                Monitor(state_file=sf)

    def test_init_surge_state_file_none_uses_default(self, tmp_path):
        """surge_state_file=None → SurgeState constructed with no kwargs."""
        from monitor import Monitor

        sf = str(tmp_path / "st.json")
        with patch.dict(os.environ, BASE_ENV, clear=False):
            with patch("monitor.XClient"), patch("monitor.SlackClient"):
                with patch("monitor.SurgeState") as MockSurge:
                    Monitor(state_file=sf)  # surge_state_file defaults to None

        MockSurge.assert_called_once_with()

    def test_init_surge_state_file_provided_passes_kwarg(self, tmp_path):
        """surge_state_file provided → SurgeState(state_file=...) is called."""
        from monitor import Monitor

        sf = str(tmp_path / "st.json")
        surge_sf = str(tmp_path / "surge.json")
        with patch.dict(os.environ, BASE_ENV, clear=False):
            with patch("monitor.XClient"), patch("monitor.SlackClient"):
                with patch("monitor.SurgeState") as MockSurge:
                    Monitor(state_file=sf, surge_state_file=surge_sf)

        MockSurge.assert_called_once_with(state_file=surge_sf)

    # ------------------------------------------------------------------
    # _load_state / _save_state
    # ------------------------------------------------------------------

    def test_load_state_no_file_returns_empty_dict(self, tmp_path):
        m = self._make_monitor(tmp_path)
        assert m._load_state() == {}

    def test_load_state_existing_file_returns_contents(self, tmp_path):
        m = self._make_monitor(tmp_path, state_data={"since_id": "XYZ"})
        assert m._load_state()["since_id"] == "XYZ"

    def test_save_state_persists_to_disk(self, tmp_path):
        m = self._make_monitor(tmp_path)
        m.state["since_id"] = "saved"
        m._save_state()
        on_disk = json.loads(open(m.state_file).read())
        assert on_disk["since_id"] == "saved"

    # ------------------------------------------------------------------
    # run_poll
    # ------------------------------------------------------------------

    def test_run_poll_first_run_uses_start_time_backfill(self, tmp_path):
        """No since_id → search_recent called with start_time set, since_id=None."""
        m = self._make_monitor(tmp_path)
        m.x_client.search_recent.return_value = [SAMPLE_TWEET]

        tweets = m.run_poll()

        kw = m.x_client.search_recent.call_args.kwargs
        assert kw["since_id"] is None
        assert kw["start_time"] is not None   # 24h backfill
        assert tweets == [SAMPLE_TWEET]
        assert m.state["since_id"] == SAMPLE_TWEET["id"]

    def test_run_poll_subsequent_run_passes_since_id(self, tmp_path):
        m = self._make_monitor(tmp_path, state_data={"since_id": "prev_tweet_id"})
        m.x_client.search_recent.return_value = [SAMPLE_TWEET]

        m.run_poll()

        kw = m.x_client.search_recent.call_args.kwargs
        assert kw["since_id"] == "prev_tweet_id"

    def test_run_poll_no_tweets_does_not_post_to_slack(self, tmp_path):
        m = self._make_monitor(tmp_path)
        m.x_client.search_recent.return_value = []

        tweets = m.run_poll()

        m.slack_client.post_mentions.assert_not_called()
        assert "since_id" not in m.state
        assert tweets == []

    def test_run_poll_no_tweets_preserves_existing_since_id(self, tmp_path):
        m = self._make_monitor(tmp_path, state_data={"since_id": "old_id"})
        m.x_client.search_recent.return_value = []

        m.run_poll()

        assert m.state["since_id"] == "old_id"

    def test_run_poll_new_tweets_posts_to_slack_and_updates_since_id(self, tmp_path):
        m = self._make_monitor(tmp_path)
        m.x_client.search_recent.return_value = [SAMPLE_TWEET]

        m.run_poll()

        m.slack_client.post_mentions.assert_called_once_with([SAMPLE_TWEET])
        assert m.state["since_id"] == SAMPLE_TWEET["id"]

    # ------------------------------------------------------------------
    # _should_send_digest
    # ------------------------------------------------------------------

    def test_should_send_digest_wrong_hour_returns_false(self, tmp_path):
        m = self._make_monitor(tmp_path)
        fake_now = datetime(2024, 1, 1, 15, 0, 0, tzinfo=timezone.utc)  # 15:00 UTC
        with patch("monitor.datetime") as mock_dt:
            mock_dt.now.return_value = fake_now
            assert m._should_send_digest() is False

    def test_should_send_digest_correct_hour_not_yet_sent_returns_true(self, tmp_path):
        m = self._make_monitor(tmp_path)
        fake_now = datetime(2024, 1, 1, 20, 0, 0, tzinfo=timezone.utc)  # 20:00 UTC
        with patch("monitor.datetime") as mock_dt:
            mock_dt.now.return_value = fake_now
            assert m._should_send_digest() is True

    def test_should_send_digest_already_sent_today_returns_false(self, tmp_path):
        m = self._make_monitor(tmp_path, state_data={"last_digest_date": "2024-01-01"})
        fake_now = datetime(2024, 1, 1, 20, 0, 0, tzinfo=timezone.utc)
        with patch("monitor.datetime") as mock_dt:
            mock_dt.now.return_value = fake_now
            assert m._should_send_digest() is False

    # ------------------------------------------------------------------
    # run_daily_digest
    # ------------------------------------------------------------------

    def test_run_daily_digest_posts_count_and_resets(self, tmp_path):
        m = self._make_monitor(tmp_path, state_data={"daily_count": 7})

        m.run_daily_digest()

        m.slack_client.post_digest.assert_called_once_with({"count": 7})
        assert m.state["daily_count"] == 0
        assert "last_digest_date" in m.state

    # ------------------------------------------------------------------
    # _run_once
    # ------------------------------------------------------------------

    def test_run_once_no_digest_returns_normal_interval(self, tmp_path):
        from monitor import POLL_INTERVAL_SECONDS

        m = self._make_monitor(tmp_path)
        m.x_client.search_recent.return_value = [SAMPLE_TWEET]

        with patch.object(m, "_should_send_digest", return_value=False):
            interval = m._run_once()

        assert m.state["daily_count"] == 1
        assert interval == POLL_INTERVAL_SECONDS

    def test_run_once_triggers_digest_when_due(self, tmp_path):
        m = self._make_monitor(tmp_path)
        m.x_client.search_recent.return_value = []

        with patch.object(m, "_should_send_digest", return_value=True):
            with patch.object(m, "run_daily_digest") as mock_digest:
                m._run_once()

        mock_digest.assert_called_once()

    def test_run_once_returns_surge_interval_when_surge_active(self, tmp_path):
        from monitor import SURGE_INTERVAL_SECONDS

        m = self._make_monitor(tmp_path)
        m.x_client.search_recent.return_value = []
        m.surge.enable(duration_hours=6)

        with patch.object(m, "_should_send_digest", return_value=False):
            interval = m._run_once()

        assert interval == SURGE_INTERVAL_SECONDS

    # ------------------------------------------------------------------
    # run (infinite loop)
    # ------------------------------------------------------------------

    def test_run_normal_path_sleeps_with_returned_interval(self, tmp_path):
        from monitor import Monitor, POLL_INTERVAL_SECONDS

        sf = str(tmp_path / "st.json")
        surge_sf = str(tmp_path / "surge.json")
        with patch.dict(os.environ, BASE_ENV, clear=False):
            with patch("monitor.XClient"), patch("monitor.SlackClient"):
                m = Monitor(state_file=sf, surge_state_file=surge_sf)

        sleep_calls = []

        def fake_sleep(n):
            sleep_calls.append(n)
            raise SystemExit("terminate test loop")

        with patch.object(m, "_run_once", return_value=POLL_INTERVAL_SECONDS):
            with patch("monitor.time.sleep", side_effect=fake_sleep):
                with pytest.raises(SystemExit):
                    m.run()

        assert sleep_calls == [POLL_INTERVAL_SECONDS]

    def test_run_exception_in_run_once_falls_back_to_poll_interval(self, tmp_path):
        from monitor import Monitor, POLL_INTERVAL_SECONDS

        sf = str(tmp_path / "st.json")
        surge_sf = str(tmp_path / "surge.json")
        with patch.dict(os.environ, BASE_ENV, clear=False):
            with patch("monitor.XClient"), patch("monitor.SlackClient"):
                m = Monitor(state_file=sf, surge_state_file=surge_sf)

        sleep_calls = []

        def fake_sleep(n):
            sleep_calls.append(n)
            raise SystemExit("terminate test loop")

        with patch.object(m, "_run_once", side_effect=RuntimeError("api exploded")):
            with patch("monitor.time.sleep", side_effect=fake_sleep):
                with pytest.raises(SystemExit):
                    m.run()

        # On exception, sleep is called with the ambient interval
        assert sleep_calls == [POLL_INTERVAL_SECONDS]
