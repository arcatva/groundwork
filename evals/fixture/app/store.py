from dataclasses import dataclass, field
from datetime import datetime


@dataclass
class User:
    id: int
    name: str
    is_admin: bool = False


@dataclass
class Export:
    user_id: int
    created_at: datetime


@dataclass
class Store:
    users: dict[int, User] = field(default_factory=dict)
    exports: list[Export] = field(default_factory=list)
