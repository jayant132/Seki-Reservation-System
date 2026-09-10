"""
Thin client for the Seki API.

This module intentionally does nothing except translate Python calls into
real HTTP requests against the Go backend and return real responses. There
is no mocked data, no local simulation of booking logic, and no fallback
sample data anywhere in this file — if the API is down, calls fail loudly,
because the entire point of this dashboard is to demonstrate the actual
backend behaving correctly under real conditions (including real
concurrency conflicts).
"""

from __future__ import annotations

import os
import re
import uuid
from dataclasses import dataclass
from datetime import datetime
from typing import Any

import requests

API_BASE_URL = os.environ.get("API_BASE_URL", "http://localhost:8080/api/v1")
# The root without /api/v1, for /status/*, /metrics which live outside the
# versioned API namespace.
API_ROOT_URL = API_BASE_URL.rsplit("/api/v1", 1)[0]

DEFAULT_TIMEOUT = 10


class SekiAPIError(Exception):
    """Raised when the API returns a non-2xx response. Carries the real
    status code and error message the backend returned, so the UI can
    show the user exactly what the server said instead of a generic
    failure."""

    def __init__(self, status_code: int, message: str):
        self.status_code = status_code
        self.message = message
        super().__init__(f"[{status_code}] {message}")


@dataclass
class AuthSession:
    token: str
    user_id: str
    email: str
    role: str


class SekiClient:
    """One instance per Streamlit session (held in st.session_state), so
    each browser tab/user has its own token without any global state."""

    def __init__(self, base_url: str = API_BASE_URL, token: str | None = None):
        self.base_url = base_url
        self.root_url = API_ROOT_URL
        self.token = token

    # --- internal helpers -------------------------------------------------

    def _headers(self, idempotency_key: str | None = None) -> dict:
        headers = {"Content-Type": "application/json"}
        if self.token:
            headers["Authorization"] = f"Bearer {self.token}"
        if idempotency_key:
            headers["Idempotency-Key"] = idempotency_key
        return headers

    def _request(self, method: str, path: str, base: str | None = None, **kwargs) -> Any:
        url = f"{base or self.base_url}{path}"
        try:
            resp = requests.request(method, url, timeout=DEFAULT_TIMEOUT, **kwargs)
        except requests.exceptions.ConnectionError as e:
            raise SekiAPIError(0, f"Could not reach the Seki API at {url}. Is `docker compose up` running? ({e})")
        except requests.exceptions.Timeout:
            raise SekiAPIError(0, f"Request to {url} timed out after {DEFAULT_TIMEOUT}s.")

        if resp.status_code >= 400:
            try:
                message = resp.json().get("error", resp.text)
            except ValueError:
                message = resp.text
            raise SekiAPIError(resp.status_code, message)

        if resp.status_code == 204 or not resp.content:
            return None
        return resp.json()

    # --- auth ---------------------------------------------------------------

    def register(self, email: str, password: str) -> AuthSession:
        data = self._request("POST", "/auth/register", json={"email": email, "password": password})
        return AuthSession(
            token=data["token"],
            user_id=data["user"]["id"],
            email=data["user"]["email"],
            role=data["user"]["role"],
        )

    def login(self, email: str, password: str) -> AuthSession:
        data = self._request("POST", "/auth/login", json={"email": email, "password": password})
        return AuthSession(
            token=data["token"],
            user_id=data["user"]["id"],
            email=data["user"]["email"],
            role=data["user"]["role"],
        )

    # --- resources ------------------------------------------------------

    def list_resources(self) -> list[dict]:
        return self._request("GET", "/resources") or []

    def create_resource(self, name: str, description: str, capacity: int) -> dict:
        return self._request(
            "POST", "/resources",
            headers=self._headers(),
            json={"name": name, "description": description, "capacity": capacity},
        )

    def availability(self, resource_id: str, from_dt: datetime, to_dt: datetime) -> list[dict]:
        return self._request(
            "GET", f"/resources/{resource_id}/availability",
            params={"from": from_dt.isoformat() + "Z", "to": to_dt.isoformat() + "Z"},
        ) or []

    # --- bookings -------------------------------------------------------

    def create_booking(
        self, resource_id: str, start_dt: datetime, end_dt: datetime,
        notes: str = "", idempotency_key: str | None = None,
    ) -> dict:
        key = idempotency_key or str(uuid.uuid4())
        return self._request(
            "POST", "/bookings",
            headers=self._headers(idempotency_key=key),
            json={
                "resource_id": resource_id,
                "start_time": start_dt.isoformat() + "Z",
                "end_time": end_dt.isoformat() + "Z",
                "notes": notes,
            },
        )

    def cancel_booking(self, booking_id: str) -> dict:
        return self._request("POST", f"/bookings/{booking_id}/cancel", headers=self._headers())

    def my_bookings(self) -> list[dict]:
        return self._request("GET", "/bookings/mine", headers=self._headers()) or []

    # --- audit ------------------------------------------------------------

    def audit_trail(self, entity_id: str) -> list[dict]:
        return self._request("GET", f"/audit/{entity_id}") or []

    def verify_audit_chain(self) -> dict:
        return self._request("GET", "/audit/verify")

    # --- system status / observability -------------------------------------

    def status_live(self) -> dict:
        return self._request("GET", "/status/live", base=self.root_url)

    def status_ready(self) -> dict:
        return self._request("GET", "/status/ready", base=self.root_url)

    def raw_metrics(self) -> str:
        """Fetches the raw Prometheus text-format /metrics output. Parsed
        client-side below rather than pulled through a second service, so
        the dashboard shows exactly what Prometheus itself is scraping."""
        resp = requests.get(f"{self.root_url}/metrics", timeout=DEFAULT_TIMEOUT)
        resp.raise_for_status()
        return resp.text

    def key_metric_counters(self) -> dict[str, float]:
        """Extracts the handful of counters worth surfacing directly in the
        dashboard, by parsing the raw Prometheus exposition format. This
        avoids adding a second dependency just to read three numbers."""
        text = self.raw_metrics()
        wanted = {
            "seki_bookings_created_total": "bookings_created",
            "seki_booking_conflicts_total": "booking_conflicts",
            "seki_bookings_cancelled_total": "bookings_cancelled",
        }
        results = {v: 0.0 for v in wanted.values()}
        for line in text.splitlines():
            if line.startswith("#"):
                continue
            for metric_name, key in wanted.items():
                if line.startswith(metric_name):
                    match = re.search(r"([0-9.eE+-]+)\s*$", line)
                    if match:
                        results[key] = float(match.group(1))
        return results
