"""Auth gate for the /transcript Starlette route.

Extracted from main.py so the security-critical logic is unit-testable
without standing up the full uvicorn/a2a/httpx import stack.

#328: the route must fail CLOSED when the expected token is unavailable
(bootstrap window, missing file, OSError). The previous implementation
treated a missing token as "skip auth entirely" — any container on the
same Docker network could read the session log during provisioning.
"""


def transcript_authorized(expected_token: str | None, auth_header: str) -> bool:
    """Return True iff /transcript should serve the request.

    Args:
        expected_token: the workspace's registered bearer token, or None
            if `/configs/.auth_token` is absent / unreadable.
        auth_header: raw value of the Authorization request header.

    Behavior:
        - None/empty expected → fail closed (401). This is the #328 fix;
          a missing token file is an auth failure, not a bypass.
        - Non-empty expected: strict equality check against "Bearer <tok>".
          Bearer prefix is case-sensitive (matches the platform's
          wsauth.BearerTokenFromHeader contract).
    """
    if not expected_token:
        return False
    return auth_header == f"Bearer {expected_token}"
