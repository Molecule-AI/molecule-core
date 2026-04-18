"""Tests for coordinator routing policy."""

import json

from policies.routing import (
    build_team_routing_payload,
    build_team_route_decision,
    decide_team_route,
    summarize_children,
    _load_agent_card,
)


def test_summarize_children_extracts_skills():
    children = [
        {
            "id": "child-1",
            "name": "Alpha",
            "status": "online",
            "agent_card": {"skills": [{"name": "research"}, {"id": "write"}]},
        }
    ]

    assert summarize_children(children) == [
        {
            "id": "child-1",
            "name": "Alpha",
            "status": "online",
            "skills": ["research", "write"],
        }
    ]


def test_build_team_routing_payload_handles_empty_children():
    payload = build_team_routing_payload([], "Investigate the issue")

    assert payload["success"] is False
    assert "No team members available" in payload["error"]


def test_decide_team_route_prefers_direct_member():
    payload = decide_team_route(
        [{"id": "child-1"}],
        task="Investigate the issue",
        preferred_member_id="child-2",
    )

    assert payload["action"] == "delegate_to_preferred_member"
    assert payload["preferred_member_id"] == "child-2"


# ---------------------------------------------------------------------------
# _load_agent_card() tests
# ---------------------------------------------------------------------------

def test_load_agent_card_valid_json_string():
    """A valid JSON string that decodes to a dict is returned as a dict."""
    card = json.dumps({"name": "Alpha", "skills": [{"name": "search"}]})
    result = _load_agent_card(card)
    assert result == {"name": "Alpha", "skills": [{"name": "search"}]}


def test_load_agent_card_invalid_json_string():
    """An invalid JSON string returns an empty dict."""
    result = _load_agent_card("{not valid json}")
    assert result == {}


def test_load_agent_card_json_string_not_dict():
    """A valid JSON string that decodes to a non-dict (e.g. a list) returns {}."""
    result = _load_agent_card(json.dumps(["item1", "item2"]))
    assert result == {}


# ---------------------------------------------------------------------------
# build_team_routing_payload() with no members
# ---------------------------------------------------------------------------

def test_build_team_routing_payload_no_children_returns_error():
    """build_team_routing_payload with empty children returns an error dict."""
    result = build_team_routing_payload([], task="Do something")
    assert result["success"] is False
    assert "error" in result
    assert "No team members available" in result["error"]
    assert result["members"] == []
    assert result["task"] == "Do something"


# ---------------------------------------------------------------------------
# build_team_route_decision() compatibility wrapper
# ---------------------------------------------------------------------------

def test_build_team_route_decision_delegates_correctly():
    """build_team_route_decision is a compatibility wrapper for build_team_routing_payload."""
    children = [
        {
            "id": "child-1",
            "name": "Worker",
            "status": "online",
            "agent_card": {"skills": [{"name": "coding"}]},
        }
    ]
    result = build_team_route_decision(children, task="Write code")
    assert result["success"] is True
    assert result["action"] == "choose_member"
    assert result["task"] == "Write code"
    assert len(result["members"]) == 1


def test_build_team_route_decision_with_preferred_member():
    """build_team_route_decision passes preferred_member_id through."""
    result = build_team_route_decision(
        [{"id": "child-1"}],
        task="Analyze data",
        preferred_member_id="child-1",
    )
    assert result["action"] == "delegate_to_preferred_member"
    assert result["preferred_member_id"] == "child-1"
