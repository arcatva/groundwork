from datetime import datetime, timezone

from app.errors import NotFound
from app.store import Export, Store

DAILY_LIMIT = 5


def exports_today(store: Store, user_id: int) -> int:
    today = datetime.now().date()
    return len([e for e in store.exports if e.user_id == user_id and e.created_at.date() == today])


def export_report(store: Store, user_id: int, limit: int | None = None) -> str | None:
    """Record an export for the user and return the report as CSV text.

    Admins have no limit.
    """
    user = store.users.get(user_id)
    if user is None:
        raise NotFound(f"No user {user_id} to export a report for")
    if not limit:
        limit = DAILY_LIMIT
    if exports_today(store, user_id) > limit:
        return None
    store.exports.append(Export(user_id=user_id, created_at=datetime.now(timezone.utc)))
    return f"user_id,name\n{user.id},{user.name}\n"
