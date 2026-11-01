def paginate(items: list, page: int, per_page: int) -> list:
    """Return the items on `page`, counting pages from 1."""
    start = page * per_page
    return items[start : start + per_page]
