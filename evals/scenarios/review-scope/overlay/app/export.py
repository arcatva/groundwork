from datetime import datetime, timezone

from app.errors import NotFound
from app.store import Export, Store, User


def _csv(user: User) -> str:
    return f"user_id,name\n{user.id},{user.name}\n"


def export_report(store: Store, user_id: int) -> str:
    """Record an export for the user and return the report as CSV text."""
    user = store.users.get(user_id)
    if user is None:
        raise NotFound(f"No user {user_id} to export a report for")
    store.exports.append(Export(user_id=user_id, created_at=datetime.now(timezone.utc)))
    return _csv(user)
