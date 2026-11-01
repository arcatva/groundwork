import unittest

from app.pagination import paginate


class PaginateTest(unittest.TestCase):
    def test_first_page_holds_the_first_items(self):
        self.assertEqual(paginate([1, 2, 3, 4, 5], 1, 2), [1, 2])
