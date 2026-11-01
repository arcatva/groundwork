from datetime import datetime, timezone

from app.errors import NotFound
from app.store import Export, Store
from app.users import find_user


def export_report(store: Store, user_id: int) -> str:
    """Record an export for the user and return the report as CSV text."""
    user = find_user(store, user_id)
    if user is None:
        raise NotFound(f"No user {user_id} to export a report for")
    store.exports.append(Export(user_id=user_id, created_at=datetime.now(timezone.utc)))
    return f"id,name\n{user.id},{user.name}\n"
