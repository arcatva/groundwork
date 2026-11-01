import unittest

from app.errors import NotFound
from app.export import export_report
from app.store import Store, User


class ExportReportTest(unittest.TestCase):
    def test_records_the_export(self):
        store = Store(users={1: User(id=1, name="Ana")})
        export_report(store, 1)
        self.assertEqual(len(store.exports), 1)

    def test_unknown_user_raises(self):
        with self.assertRaises(NotFound):
            export_report(Store(), 9)
