#!/usr/bin/env python3
"""
Unit tests for issue dependency parsing, evaluation, and PR blocking logic.
"""

import os
import sys
import unittest
from unittest.mock import MagicMock

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from manage_issue_dependencies import (
    BLOCKED_LABEL,
    evaluate_dependencies,
    handle_issue_event,
    handle_pull_request_event,
    parse_dependencies_from_text,
    parse_linked_issues,
    update_issue_blocked_status,
)


class TestIssueDependencyParsing(unittest.TestCase):
    def test_single_blocked_by(self):
        text = "This issue is blocked by #13."
        deps = parse_dependencies_from_text(text, self_number=14)
        self.assertEqual(deps, [13])

    def test_multiple_blocked_by(self):
        text = "Blocked by #13, #14 and #15."
        deps = parse_dependencies_from_text(text, self_number=20)
        self.assertEqual(sorted(deps), [13, 14, 15])

    def test_colon_blocked_by(self):
        text = """
### Context
Some info here.

blocked-by: #13, #42
blocking: none
"""
        deps = parse_dependencies_from_text(text, self_number=99)
        self.assertEqual(sorted(deps), [13, 42])

    def test_depends_on(self):
        text = "Depends on #5 and depends-on: #7"
        deps = parse_dependencies_from_text(text, self_number=99)
        self.assertEqual(sorted(deps), [5, 7])

    def test_parent_issue(self):
        text = "Parent issue: #10\nSub-issue of #12"
        deps = parse_dependencies_from_text(text, self_number=99)
        self.assertEqual(sorted(deps), [10, 12])

    def test_full_github_url(self):
        text = "Blocked by https://github.com/wanjiru-noreen/halisi/issues/25."
        deps = parse_dependencies_from_text(text, self_number=99)
        self.assertEqual(deps, [25])

    def test_dependencies_section(self):
        text = """
### Dependencies
- #13
- #14
- [ ] #15
- [x] #16
"""
        deps = parse_dependencies_from_text(text, self_number=99)
        self.assertEqual(sorted(deps), [13, 14, 15, 16])

    def test_ignore_self_reference(self):
        text = "Blocked by #14, #15"
        deps = parse_dependencies_from_text(text, self_number=14)
        self.assertEqual(deps, [15])

    def test_no_dependencies(self):
        text = "Fixed typo in README #123 is just a mention."
        deps = parse_dependencies_from_text(text, self_number=99)
        self.assertEqual(deps, [])

    def test_parse_linked_issues(self):
        text = "Closes #14, fixes #20 and Resolves #25"
        linked = parse_linked_issues(text)
        self.assertEqual(sorted(linked), [14, 20, 25])


class TestDependencyEvaluation(unittest.TestCase):
    def setUp(self):
        self.mock_client = MagicMock()
        self.mock_client.get_native_dependencies.return_value = []

    def test_issue_blocked_by_open_dependency(self):
        # Issue 14 blocked by Issue 13
        self.mock_client.get_issue.side_effect = lambda num: {
            "number": num,
            "state": "open" if num == 13 else "open",
        }
        body = "Blocked by #13"
        is_blocked, open_blockers, closed_blockers = evaluate_dependencies(
            self.mock_client, 14, body
        )
        self.assertTrue(is_blocked)
        self.assertEqual(open_blockers, [13])
        self.assertEqual(closed_blockers, [])

    def test_issue_unblocked_when_dependency_closed(self):
        # Issue 14 depends on Issue 13 which is closed
        self.mock_client.get_issue.side_effect = lambda num: {
            "number": num,
            "state": "closed" if num == 13 else "open",
        }
        body = "Blocked by #13"
        is_blocked, open_blockers, closed_blockers = evaluate_dependencies(
            self.mock_client, 14, body
        )
        self.assertFalse(is_blocked)
        self.assertEqual(open_blockers, [])
        self.assertEqual(closed_blockers, [13])

    def test_label_applied_when_blocked(self):
        self.mock_client.get_issue.side_effect = lambda num: {
            "number": num,
            "state": "open",
        }
        issue_data = {
            "number": 14,
            "body": "Blocked by #13",
            "labels": [],
        }
        is_blocked = update_issue_blocked_status(self.mock_client, issue_data, notify=True)
        self.assertTrue(is_blocked)
        self.mock_client.add_label.assert_called_once_with(14, BLOCKED_LABEL)
        self.mock_client.add_comment.assert_called_once()

    def test_label_removed_when_unblocked(self):
        self.mock_client.get_issue.side_effect = lambda num: {
            "number": num,
            "state": "closed",
        }
        issue_data = {
            "number": 14,
            "body": "Blocked by #13",
            "labels": [{"name": BLOCKED_LABEL}],
        }
        is_blocked = update_issue_blocked_status(self.mock_client, issue_data, notify=True)
        self.assertFalse(is_blocked)
        self.mock_client.remove_label.assert_called_once_with(14, BLOCKED_LABEL)
        self.mock_client.add_comment.assert_called_once()

    def test_pr_fails_when_closing_blocked_issue(self):
        # PR closes #14, but #14 is blocked by open #13
        def get_issue_mock(num):
            if num == 14:
                return {"number": 14, "state": "open", "body": "Blocked by #13"}
            if num == 13:
                return {"number": 13, "state": "open", "body": ""}
            return None

        self.mock_client.get_issue.side_effect = get_issue_mock

        event_data = {
            "pull_request": {
                "number": 65,
                "title": "feat: normalize schema",
                "body": "Closes #14",
                "labels": [],
            }
        }
        exit_code = handle_pull_request_event(self.mock_client, event_data)
        self.assertEqual(exit_code, 1)
        self.mock_client.add_label.assert_called_once_with(65, BLOCKED_LABEL)

    def test_pr_passes_when_closing_unblocked_issue(self):
        # PR closes #14, and #14 has no blockers or all blockers are closed
        def get_issue_mock(num):
            if num == 14:
                return {"number": 14, "state": "open", "body": "Blocked by #13"}
            if num == 13:
                return {"number": 13, "state": "closed", "body": ""}
            return None

        self.mock_client.get_issue.side_effect = get_issue_mock

        event_data = {
            "pull_request": {
                "number": 65,
                "title": "feat: normalize schema",
                "body": "Closes #14",
                "labels": [],
            }
        }
        exit_code = handle_pull_request_event(self.mock_client, event_data)
        self.assertEqual(exit_code, 0)


if __name__ == "__main__":
    unittest.main()
