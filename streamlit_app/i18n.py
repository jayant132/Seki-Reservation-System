"""
Minimal EN/JA label dictionary for the Streamlit demo layer.

This is deliberately simple (a flat dict, no translation service) —
the goal is to make the demo directly legible to a Japanese-speaking
reviewer, not to build a full i18n framework for a thin ops dashboard.
"""

STRINGS = {
    "app_title": {
        "en": "Seki \u2014 Reservation Engine",
        "ja": "Seki \uff0d \u4e88\u7d04\u30a8\u30f3\u30b8\u30f3",
    },
    "app_subtitle": {
        "en": "A thin demo layer over the real Go/PostgreSQL backend. Every action below is a live HTTP call \u2014 nothing here is mocked.",
        "ja": "Go/PostgreSQL\u30d0\u30c3\u30af\u30a8\u30f3\u30c9\u4e0a\u306e\u8584\u3044\u30c7\u30e2\u5c64\u3067\u3059\u3002\u4ee5\u4e0b\u306e\u64cd\u4f5c\u306f\u3059\u3079\u3066\u5b9f\u969b\u306eHTTP\u547c\u3073\u51fa\u3057\u3067\u3042\u308a\u3001\u30e2\u30c3\u30af\u30c7\u30fc\u30bf\u306f\u4e00\u5207\u4f7f\u7528\u3057\u3066\u3044\u307e\u305b\u3093\u3002",
    },
    "language": {"en": "Language", "ja": "\u8a00\u8a9e"},
    "login": {"en": "Log in", "ja": "\u30ed\u30b0\u30a4\u30f3"},
    "register": {"en": "Register", "ja": "\u65b0\u898f\u767b\u9332"},
    "logout": {"en": "Log out", "ja": "\u30ed\u30b0\u30a2\u30a6\u30c8"},
    "email": {"en": "Email", "ja": "\u30e1\u30fc\u30eb\u30a2\u30c9\u30ec\u30b9"},
    "password": {"en": "Password", "ja": "\u30d1\u30b9\u30ef\u30fc\u30c9"},
    "logged_in_as": {"en": "Logged in as", "ja": "\u30ed\u30b0\u30a4\u30f3\u4e2d\uff1a"},
    "role": {"en": "Role", "ja": "\u6a29\u9650"},
    "tab_book": {"en": "Book a Resource", "ja": "\u4e88\u7d04\u3059\u308b"},
    "tab_mine": {"en": "My Bookings", "ja": "\u81ea\u5206\u306e\u4e88\u7d04"},
    "tab_audit": {"en": "Audit Trail", "ja": "\u76e3\u67fb\u30ed\u30b0"},
    "tab_race": {"en": "Concurrency Demo", "ja": "\u540c\u6642\u4e88\u7d04\u30c7\u30e2"},
    "tab_status": {"en": "System Status", "ja": "\u30b7\u30b9\u30c6\u30e0\u72b6\u614b"},
    "resources": {"en": "Bookable resources", "ja": "\u4e88\u7d04\u53ef\u80fd\u306a\u30ea\u30bd\u30fc\u30b9"},
    "select_resource": {"en": "Resource", "ja": "\u30ea\u30bd\u30fc\u30b9"},
    "start_time": {"en": "Start time (JST)", "ja": "\u958b\u59cb\u6642\u523b\uff08JST\uff09"},
    "end_time": {"en": "End time (JST)", "ja": "\u7d42\u4e86\u6642\u523b\uff08JST\uff09"},
    "notes": {"en": "Notes", "ja": "\u5099\u8003"},
    "create_booking": {"en": "Create booking", "ja": "\u4e88\u7d04\u3092\u4f5c\u6210"},
    "booking_success": {"en": "Booking confirmed", "ja": "\u4e88\u7d04\u304c\u78ba\u5b9a\u3057\u307e\u3057\u305f"},
    "booking_conflict": {
        "en": "That slot was just taken \u2014 the database's exclusion constraint rejected the overlapping booking. This is the expected, correct behavior.",
        "ja": "\u305d\u306e\u6642\u9593\u5e2f\u306f\u3059\u3067\u306b\u4e88\u7d04\u6e08\u307f\u3067\u3059\u3002\u30c7\u30fc\u30bf\u30d9\u30fc\u30b9\u306e\u6392\u4ed6\u5236\u7d04\u306b\u3088\u308a\u91cd\u8907\u4e88\u7d04\u304c\u6b63\u3057\u304f\u62d2\u5426\u3055\u308c\u307e\u3057\u305f\uff08\u4ed5\u69d8\u901a\u308a\u306e\u52d5\u4f5c\u3067\u3059\uff09\u3002",
    },
    "cancel": {"en": "Cancel", "ja": "\u30ad\u30e3\u30f3\u30bb\u30eb"},
    "no_bookings": {"en": "No bookings yet.", "ja": "\u4e88\u7d04\u306f\u307e\u3060\u3042\u308a\u307e\u305b\u3093\u3002"},
    "audit_entity_id": {"en": "Booking ID", "ja": "\u4e88\u7d04ID"},
    "load_trail": {"en": "Load audit trail", "ja": "\u76e3\u67fb\u30ed\u30b0\u3092\u8868\u793a"},
    "verify_chain": {"en": "Verify entire audit chain integrity", "ja": "\u76e3\u67fb\u30c1\u30a7\u30fc\u30f3\u5168\u4f53\u306e\u6574\u5408\u6027\u3092\u691c\u8a3c"},
    "chain_valid": {"en": "Audit chain is intact \u2014 no entries have been altered.", "ja": "\u76e3\u67fb\u30c1\u30a7\u30fc\u30f3\u306f\u6b63\u5e38\u3067\u3059\uff08\u6539\u3056\u3093\u306a\u3057\uff09\u3002"},
    "chain_broken": {"en": "Chain integrity check FAILED at entry", "ja": "\u76e3\u67fb\u30c1\u30a7\u30fc\u30f3\u306e\u6574\u5408\u6027\u691c\u8a3c\u306b\u5931\u6557\u3057\u307e\u3057\u305f\uff08\u30a8\u30f3\u30c8\u30ea\uff09"},
    "race_intro": {
        "en": "This fires several concurrent booking requests at the exact same resource and time slot, straight at the live API \u2014 the same test the automated load test runs, but click-to-run. It exists to make the core guarantee of this project visible: exactly one request should win, and every other one should be cleanly rejected with a conflict, never a duplicate.",
        "ja": "\u3053\u306e\u30c7\u30e2\u306f\u3001\u540c\u4e00\u306e\u30ea\u30bd\u30fc\u30b9\u30fb\u540c\u4e00\u306e\u6642\u9593\u5e2f\u306b\u5bfe\u3057\u3066\u8907\u6570\u306e\u4e88\u7d04\u30ea\u30af\u30a8\u30b9\u30c8\u3092\u540c\u6642\u306b\u9001\u4fe1\u3057\u307e\u3059\u3002\u81ea\u52d5\u8ca0\u8377\u30c6\u30b9\u30c8\u3068\u540c\u3058\u691c\u8a3c\u3092\u30af\u30ea\u30c3\u30af\u4e00\u3064\u3067\u5b9f\u884c\u3067\u304d\u307e\u3059\u3002\u672c\u30d7\u30ed\u30b8\u30a7\u30af\u30c8\u306e\u4e2d\u5fc3\u7684\u306a\u4fdd\u8a3c\uff08\u91cd\u8907\u4e88\u7d04\u30bc\u30ed\uff09\u3092\u76ee\u3067\u78ba\u8a8d\u3067\u304d\u307e\u3059\u3002",
    },
    "race_run": {"en": "Fire concurrent booking requests", "ja": "\u540c\u6642\u4e88\u7d04\u30ea\u30af\u30a8\u30b9\u30c8\u3092\u5b9f\u884c"},
    "race_concurrency": {"en": "Number of concurrent requests", "ja": "\u540c\u6642\u30ea\u30af\u30a8\u30b9\u30c8\u6570"},
    "race_result_success": {"en": "Succeeded (201)", "ja": "\u6210\u529f\uff08201\uff09"},
    "race_result_conflict": {"en": "Correctly rejected (409)", "ja": "\u6b63\u3057\u304f\u62d2\u5426\uff08409\uff09"},
    "race_result_unexpected": {"en": "Unexpected errors", "ja": "\u4e88\u671f\u3057\u306a\u3044\u30a8\u30e9\u30fc"},
    "status_live": {"en": "Process liveness (/status/live)", "ja": "\u30d7\u30ed\u30bb\u30b9\u306e\u751f\u5b58\u78ba\u8a8d\uff08/status/live\uff09"},
    "status_ready": {"en": "Dependency readiness (/status/ready)", "ja": "\u4f9d\u5b58\u30b5\u30fc\u30d3\u30b9\u306e\u6e96\u5099\u78ba\u8a8d\uff08/status/ready\uff09"},
    "metrics_title": {"en": "Live metrics (parsed from /metrics)", "ja": "\u30e9\u30a4\u30d6\u30e1\u30c8\u30ea\u30af\u30b9\uff08/metrics\u3088\u308a\u53d6\u5f97\uff09"},
    "observability_links": {"en": "Observability dashboards", "ja": "\u30aa\u30d6\u30b6\u30d0\u30d3\u30ea\u30c6\u30a3\u30c0\u30c3\u30b7\u30e5\u30dc\u30fc\u30c9"},
    "admin_create_resource": {"en": "Admin: create a new resource", "ja": "\u7ba1\u7406\u8005\uff1a\u65b0\u3057\u3044\u30ea\u30bd\u30fc\u30b9\u3092\u4f5c\u6210"},
    "resource_name": {"en": "Name", "ja": "\u540d\u524d"},
    "resource_description": {"en": "Description", "ja": "\u8aac\u660e"},
    "resource_capacity": {"en": "Capacity", "ja": "\u5b9a\u54e1"},
    "create": {"en": "Create", "ja": "\u4f5c\u6210"},
    "please_login": {"en": "Log in or register from the sidebar to continue.", "ja": "\u5229\u7528\u3059\u308b\u306b\u306f\u30b5\u30a4\u30c9\u30d0\u30fc\u304b\u3089\u30ed\u30b0\u30a4\u30f3\u307e\u305f\u306f\u767b\u9332\u3057\u3066\u304f\u3060\u3055\u3044\u3002",},
}


def t(key: str, lang: str) -> str:
    entry = STRINGS.get(key)
    if not entry:
        return key
    return entry.get(lang, entry.get("en", key))
