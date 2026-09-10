"""
Seki demo dashboard.

A thin Streamlit layer over the real Go/PostgreSQL backend. It exists to
make the project's behavior visible to a non-terminal audience (a
recruiter, a hiring manager, a non-technical reviewer) without adding any
business logic of its own — every button here triggers a real HTTP call
against the live API defined in api_client.py, including the concurrency
demo, which fires real concurrent requests and shows the real result.

Run via `docker compose up` (recommended) or standalone:
    pip install -r requirements.txt
    streamlit run app.py
"""

from __future__ import annotations

import os
from concurrent.futures import ThreadPoolExecutor, as_completed
from datetime import datetime, timedelta

import pandas as pd
import streamlit as st

from api_client import SekiAPIError, SekiClient
from i18n import t

st.set_page_config(page_title="Seki \u2014 Reservation Engine", page_icon="\U0001F4C5", layout="wide")

JAEGER_UI_URL = os.environ.get("JAEGER_UI_URL", "http://localhost:16686")
PROMETHEUS_UI_URL = os.environ.get("PROMETHEUS_UI_URL", "http://localhost:9090")
GRAFANA_UI_URL = os.environ.get("GRAFANA_UI_URL", "http://localhost:3000")

# --- session state init --------------------------------------------------

if "lang" not in st.session_state:
    st.session_state.lang = "en"
if "session" not in st.session_state:
    st.session_state.session = None  # AuthSession once logged in

lang = st.session_state.lang
client = SekiClient(token=st.session_state.session.token if st.session_state.session else None)


def _(key: str) -> str:
    return t(key, lang)


# --- sidebar: language + auth --------------------------------------------

with st.sidebar:
    st.selectbox(
        _("language"), options=["en", "ja"],
        format_func=lambda x: "English" if x == "en" else "\u65e5\u672c\u8a9e",
        key="lang",
    )
    st.divider()

    if st.session_state.session:
        sess = st.session_state.session
        st.markdown(f"**{_('logged_in_as')}** {sess.email}")
        st.caption(f"{_('role')}: {sess.role}")
        if st.button(_("logout"), use_container_width=True):
            st.session_state.session = None
            st.rerun()
    else:
        auth_tab_login, auth_tab_register = st.tabs([_("login"), _("register")])

        with auth_tab_login:
            with st.form("login_form"):
                email = st.text_input(_("email"), key="login_email")
                password = st.text_input(_("password"), type="password", key="login_password")
                if st.form_submit_button(_("login"), use_container_width=True):
                    try:
                        st.session_state.session = client.login(email, password)
                        st.rerun()
                    except SekiAPIError as e:
                        st.error(f"{e.status_code}: {e.message}")

        with auth_tab_register:
            with st.form("register_form"):
                email = st.text_input(_("email"), key="reg_email")
                password = st.text_input(_("password"), type="password", key="reg_password",
                                          help="Minimum 8 characters")
                if st.form_submit_button(_("register"), use_container_width=True):
                    try:
                        st.session_state.session = client.register(email, password)
                        st.rerun()
                    except SekiAPIError as e:
                        st.error(f"{e.status_code}: {e.message}")

    st.divider()
    st.caption("Seeded admin: `admin@seki.dev` / `AdminPass123` (run `make seed` first)")

# --- header ---------------------------------------------------------------

st.title(_("app_title"))
st.caption(_("app_subtitle"))

# --- main content -----------------------------------------------------

tab_book, tab_mine, tab_race, tab_audit, tab_status = st.tabs([
    _("tab_book"), _("tab_mine"), _("tab_race"), _("tab_audit"), _("tab_status"),
])

# ===== Tab: Book a Resource =====
with tab_book:
    try:
        resources = client.list_resources()
    except SekiAPIError as e:
        st.error(f"{e.status_code}: {e.message}")
        resources = []

    st.subheader(_("resources"))
    if resources:
        st.dataframe(
            pd.DataFrame(resources)[["name", "description", "capacity"]],
            use_container_width=True, hide_index=True,
        )
    else:
        st.info("No resources yet. Run `make seed` or create one below as admin.")

    if st.session_state.session and st.session_state.session.role == "admin":
        with st.expander(_("admin_create_resource")):
            with st.form("create_resource_form"):
                name = st.text_input(_("resource_name"))
                description = st.text_input(_("resource_description"))
                capacity = st.number_input(_("resource_capacity"), min_value=1, value=1)
                if st.form_submit_button(_("create")):
                    try:
                        client.create_resource(name, description, int(capacity))
                        st.success("Created.")
                        st.rerun()
                    except SekiAPIError as e:
                        st.error(f"{e.status_code}: {e.message}")

    st.divider()

    if not st.session_state.session:
        st.info(_("please_login"))
    elif resources:
        resource_options = {r["name"]: r["id"] for r in resources}
        with st.form("create_booking_form"):
            resource_name = st.selectbox(_("select_resource"), options=list(resource_options.keys()))
            col1, col2 = st.columns(2)
            with col1:
                start_date = st.date_input(_("start_time"), value=datetime.utcnow().date() + timedelta(days=1))
                start_hour = st.number_input("Hour", min_value=0, max_value=23, value=10, key="start_hour")
            with col2:
                end_date = st.date_input(_("end_time"), value=datetime.utcnow().date() + timedelta(days=1))
                end_hour = st.number_input("Hour", min_value=0, max_value=23, value=11, key="end_hour")
            notes = st.text_input(_("notes"))

            if st.form_submit_button(_("create_booking"), use_container_width=True):
                start_dt = datetime.combine(start_date, datetime.min.time()) + timedelta(hours=int(start_hour))
                end_dt = datetime.combine(end_date, datetime.min.time()) + timedelta(hours=int(end_hour))
                try:
                    booking = client.create_booking(
                        resource_options[resource_name], start_dt, end_dt, notes,
                    )
                    st.success(f"{_('booking_success')} \u2014 ID: `{booking['id']}`")
                except SekiAPIError as e:
                    if e.status_code == 409:
                        st.warning(_("booking_conflict"))
                    else:
                        st.error(f"{e.status_code}: {e.message}")

# ===== Tab: My Bookings =====
with tab_mine:
    if not st.session_state.session:
        st.info(_("please_login"))
    else:
        try:
            bookings = client.my_bookings()
        except SekiAPIError as e:
            st.error(f"{e.status_code}: {e.message}")
            bookings = []

        if not bookings:
            st.info(_("no_bookings"))
        else:
            for b in bookings:
                cols = st.columns([3, 2, 2, 2, 1])
                cols[0].write(f"**{b.get('notes') or '(no notes)'}**")
                cols[1].write(b["start_time"][:16].replace("T", " "))
                cols[2].write(b["end_time"][:16].replace("T", " "))
                status_color = "\U0001F7E2" if b["status"] == "confirmed" else "\u26aa"
                cols[3].write(f"{status_color} {b['status']}")
                if b["status"] == "confirmed":
                    if cols[4].button(_("cancel"), key=f"cancel_{b['id']}"):
                        try:
                            client.cancel_booking(b["id"])
                            st.rerun()
                        except SekiAPIError as e:
                            st.error(f"{e.status_code}: {e.message}")

# ===== Tab: Concurrency Demo =====
with tab_race:
    st.write(_("race_intro"))

    if not st.session_state.session:
        st.info(_("please_login"))
    else:
        try:
            resources = client.list_resources()
        except SekiAPIError as e:
            st.error(f"{e.status_code}: {e.message}")
            resources = []

        if resources:
            resource_options = {r["name"]: r["id"] for r in resources}
            resource_name = st.selectbox(_("select_resource"), options=list(resource_options.keys()), key="race_resource")
            concurrency = st.slider(_("race_concurrency"), min_value=2, max_value=30, value=10)

            if st.button(_("race_run"), type="primary"):
                # Every worker races for the exact same far-future slot, using
                # the SAME logged-in session's token but a UNIQUE idempotency
                # key each, so this is a genuine "N different requests, one
                # slot" race rather than N idempotent retries of one request.
                resource_id = resource_options[resource_name]
                slot_start = datetime.utcnow() + timedelta(days=365)
                slot_end = slot_start + timedelta(hours=1)

                results = {"success": 0, "conflict": 0, "unexpected": 0}
                rows = []

                def attempt(i: int):
                    worker_client = SekiClient(token=client.token)
                    try:
                        b = worker_client.create_booking(resource_id, slot_start, slot_end, notes=f"racer-{i}")
                        return ("success", i, b["id"])
                    except SekiAPIError as e:
                        if e.status_code == 409:
                            return ("conflict", i, e.message)
                        return ("unexpected", i, f"{e.status_code}: {e.message}")

                with st.spinner("Firing concurrent requests..."):
                    with ThreadPoolExecutor(max_workers=concurrency) as pool:
                        futures = [pool.submit(attempt, i) for i in range(concurrency)]
                        for fut in as_completed(futures):
                            outcome, i, detail = fut.result()
                            results[outcome] += 1
                            rows.append({"request #": i, "outcome": outcome, "detail": detail})

                col1, col2, col3 = st.columns(3)
                col1.metric(_("race_result_success"), results["success"])
                col2.metric(_("race_result_conflict"), results["conflict"])
                col3.metric(_("race_result_unexpected"), results["unexpected"])

                if results["success"] == 1 and results["unexpected"] == 0:
                    st.success("Correct: exactly one request won the slot, every other one was cleanly rejected.")
                elif results["success"] > 1:
                    st.error("Unexpected: more than one booking succeeded for the same slot \u2014 this would indicate a real bug.")

                st.dataframe(pd.DataFrame(rows).sort_values("request #"), use_container_width=True, hide_index=True)
        else:
            st.info("No resources available \u2014 run `make seed` first.")

# ===== Tab: Audit Trail =====
with tab_audit:
    entity_id = st.text_input(_("audit_entity_id"))
    col1, col2 = st.columns(2)
    with col1:
        if st.button(_("load_trail")) and entity_id:
            try:
                entries = client.audit_trail(entity_id)
                if entries:
                    st.dataframe(pd.DataFrame(entries), use_container_width=True, hide_index=True)
                else:
                    st.info("No audit entries found for that ID.")
            except SekiAPIError as e:
                st.error(f"{e.status_code}: {e.message}")
    with col2:
        if st.button(_("verify_chain")):
            try:
                result = client.verify_audit_chain()
                if result.get("valid"):
                    st.success(_("chain_valid"))
                else:
                    st.error(f"{_('chain_broken')} #{result.get('broken_at_entry_id')}")
            except SekiAPIError as e:
                st.error(f"{e.status_code}: {e.message}")

# ===== Tab: System Status / Observability =====
with tab_status:
    col1, col2 = st.columns(2)
    with col1:
        st.subheader(_("status_live"))
        try:
            st.json(client.status_live())
        except SekiAPIError as e:
            st.error(f"{e.status_code}: {e.message}")
    with col2:
        st.subheader(_("status_ready"))
        try:
            ready = client.status_ready()
            st.json(ready)
        except SekiAPIError as e:
            st.error(f"{e.status_code}: {e.message}")

    st.divider()
    st.subheader(_("metrics_title"))
    try:
        counters = client.key_metric_counters()
        c1, c2, c3 = st.columns(3)
        c1.metric("Bookings created (total)", int(counters["bookings_created"]))
        c2.metric("Conflicts rejected (total)", int(counters["booking_conflicts"]))
        c3.metric("Bookings cancelled (total)", int(counters["bookings_cancelled"]))
    except Exception as e:
        st.error(f"Could not fetch /metrics: {e}")

    st.divider()
    st.subheader(_("observability_links"))
    st.markdown(
        f"- [Jaeger \u2014 distributed traces]({JAEGER_UI_URL})\n"
        f"- [Prometheus \u2014 raw metrics & queries]({PROMETHEUS_UI_URL})\n"
        f"- [Grafana \u2014 dashboards]({GRAFANA_UI_URL}) (admin/admin)"
    )
