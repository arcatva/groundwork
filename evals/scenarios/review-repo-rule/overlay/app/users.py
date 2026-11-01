from app.store import Store, User


def find_user(store: Store, user_id: int) -> User | None:
    """Return the user, or None when there is no such user."""
    return store.users.get(user_id)
